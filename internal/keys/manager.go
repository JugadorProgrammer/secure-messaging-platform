// Package keys управляет ключами RSA пользователей и системным ключом
// подписи (user_id = 0).
//
// Ключ генерируется один раз, в фоне: RSA-32768 считается десятки минут, и
// запрос пользователя не может столько ждать. Пока ключ в статусе generating,
// вызывающий отвечает 409 key_generating (для системного ключа — 503).
package keys

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// SystemUserID — владелец системного ключа, которым подписываются PDF.
const SystemUserID int64 = 0

const (
	// staleGrace — запас сверх таймаута генерации, после которого запись
	// generating считается брошенной. Без запаса перезапуск мог бы начаться,
	// пока законная генерация ещё сохраняет результат.
	staleGrace = time.Minute

	// failedWriteTimeout ограничивает запись статуса failed. Контекст генерации
	// к этому моменту может уже истечь, а статус записать нужно.
	failedWriteTimeout = 10 * time.Second
)

// Store — операции хранилища, нужные менеджеру. store.Store реализует его;
// интерфейс объявлен у потребителя, в общий ports.Store методы не выносятся.
type Store interface {
	// UserKey возвращает ports.ErrNotFound, если записи нет.
	UserKey(ctx context.Context, userID int64) (*ports.UserKey, error)
	SaveUserKey(ctx context.Context, k *ports.UserKey) error
	// StartUserKey создаёт запись generating, если её нет; started получает
	// только один из конкурирующих вызовов.
	StartUserKey(ctx context.Context, userID int64, bits int) (started bool, err error)
	// RestartUserKey переводит failed или зависший generating обратно
	// в generating; started получает только один из конкурирующих вызовов.
	RestartUserKey(ctx context.Context, userID int64, bits int, staleBefore time.Time) (started bool, err error)
}

// Config — параметры менеджера из конфигурации сервиса.
type Config struct {
	// MasterKeyHex — MASTER_KEY в hex, под ним запечатывается закрытый ключ.
	MasterKeyHex string
	// Bits — RSA_BITS, размер новых ключей.
	Bits int
	// KeygenTimeout — RSA_KEYGEN_TIMEOUT, верхняя граница одной генерации.
	KeygenTimeout time.Duration
}

// Manager создаёт ключи RSA и следит, чтобы для пользователя одновременно
// шла не больше одной генерации. Безопасен для конкурентного использования.
type Manager struct {
	store         Store
	crypto        ports.CryptoClient
	masterKeyHex  string
	bits          int
	keygenTimeout time.Duration
	log           *slog.Logger
	now           func() time.Time

	// closing закрывается в Close: генерации, начатые до остановки сервиса,
	// прерываются и помечаются failed, а не остаются в generating.
	// Канал, а не контекст: контекст в структуре не хранится (styleguide.md, раздел 7).
	closing   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
	mu        sync.Mutex
	running   map[int64]struct{}
}

// New создаёт менеджер. Генераций он не запускает: они начинаются
// в EnsureUserKey.
func New(store Store, crypto ports.CryptoClient, cfg Config, log *slog.Logger) *Manager {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	return &Manager{
		store:         store,
		crypto:        crypto,
		masterKeyHex:  cfg.MasterKeyHex,
		bits:          cfg.Bits,
		keygenTimeout: cfg.KeygenTimeout,
		log:           log,
		now:           time.Now,
		closing:       make(chan struct{}),
		running:       make(map[int64]struct{}),
	}
}

// EnsureUserKey возвращает ключ пользователя и при необходимости запускает
// его генерацию в фоне:
//
//   - ready — ключ возвращается, генерация не запускается;
//   - generating — ключ возвращается как есть, вторая генерация не запускается;
//   - failed или брошенный generating — генерация запускается заново;
//   - записи нет — создаётся запись generating и запускается генерация.
//
// Возвращается актуальная запись из базы, обычно в статусе generating:
// дожидаться готовности вызывающий не должен.
func (m *Manager) EnsureUserKey(ctx context.Context, userID int64) (*ports.UserKey, error) {
	key, err := m.store.UserKey(ctx, userID)

	switch {
	case errors.Is(err, ports.ErrNotFound):
		started, err := m.store.StartUserKey(ctx, userID, m.bits)
		if err != nil {
			return nil, fmt.Errorf("keys: start key %d: %w", userID, err)
		}

		if started {
			m.launch(ctx, userID)
		}
	case err != nil:
		return nil, fmt.Errorf("keys: get key %d: %w", userID, err)
	case key.Status == ports.KeyStatusReady:
		return key, nil
	case m.needsRestart(key):
		started, err := m.store.RestartUserKey(ctx, userID, m.bits, m.staleBefore())
		if err != nil {
			return nil, fmt.Errorf("keys: restart key %d: %w", userID, err)
		}

		if started {
			m.launch(ctx, userID)
		}
	default:
		return key, nil
	}

	key, err = m.store.UserKey(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("keys: get key %d: %w", userID, err)
	}

	return key, nil
}

// Close прерывает идущие генерации и ждёт, пока они запишут статус failed.
// После Close менеджер использовать нельзя.
// Повторный вызов безопасен.
func (m *Manager) Close() {
	m.closeOnce.Do(func() { close(m.closing) })
	m.wg.Wait()
}

// needsRestart: failed перезапускается всегда, generating — только если
// генерация брошена. Генерация, идущая в этом процессе, брошенной не бывает.
func (m *Manager) needsRestart(key *ports.UserKey) bool {
	switch key.Status {
	case ports.KeyStatusFailed:
		return !m.isRunning(key.UserID)
	case ports.KeyStatusGenerating:
		return key.UpdatedAt.Before(m.staleBefore()) && !m.isRunning(key.UserID)
	default:
		return false
	}
}

func (m *Manager) staleBefore() time.Time {
	return m.now().Add(-(m.keygenTimeout + staleGrace))
}

func (m *Manager) isRunning(userID int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.running[userID]

	return ok
}

// launch запускает генерацию, если в этом процессе она для userID ещё не идёт.
// Между репликами дубли отсекает база (StartUserKey, RestartUserKey), внутри
// процесса — карта running.
func (m *Manager) launch(ctx context.Context, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.running[userID]; ok {
		return
	}

	m.running[userID] = struct{}{}
	m.wg.Add(1)

	go m.generate(ctx, userID)
}

// generate выполняется в фоне. Контекст запроса отвязан от отмены
// (context.WithoutCancel): если пользователь закрыл вкладку, ключ всё равно
// нужен. Значения контекста, например X-Request-ID, сохраняются. Генерацию
// ограничивают RSA_KEYGEN_TIMEOUT и остановка сервиса.
func (m *Manager) generate(parent context.Context, userID int64) {
	defer m.wg.Done()
	defer m.finish(userID)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), m.keygenTimeout)
	defer cancel()

	m.cancelOnClose(ctx, cancel)

	start := m.now()

	key, err := m.createKey(ctx, userID)
	if err == nil {
		err = m.store.SaveUserKey(ctx, key)
	}

	if err != nil {
		// Текст ошибки не содержит ключей: cryptoclient и sealed не включают
		// значения полей в сообщения.
		m.log.ErrorContext(ctx, "rsa key generation failed", "user_id", userID, "bits", m.bits, "err", err)
		m.markFailed(context.WithoutCancel(ctx), userID)

		return
	}

	m.log.InfoContext(ctx, "rsa key generated",
		"user_id", userID, "bits", m.bits, "keygen_ms", key.KeygenMs, "duration", m.now().Sub(start))
}

// cancelOnClose отменяет ctx при Close. Горутина завершается вместе с ctx:
// generate отменяет его при выходе.
func (m *Manager) cancelOnClose(ctx context.Context, cancel context.CancelFunc) {
	go func() {
		select {
		case <-m.closing:
			cancel()
		case <-ctx.Done():
		}
	}()
}

// createKey генерирует пару ключей и запечатывает закрытый ключ под MASTER_KEY.
func (m *Manager) createKey(ctx context.Context, userID int64) (*ports.UserKey, error) {
	publicPEM, privatePEM, keygenMs, err := m.crypto.RSAKeys(ctx, m.bits)
	if err != nil {
		return nil, fmt.Errorf("keys: generate key %d: %w", userID, err)
	}

	// Буфер с закрытым ключом обнуляется сразу после запечатывания
	// (styleguide.md, раздел 9). Строку privatePEM, которую вернул клиент,
	// обнулить нельзя: строки в Go неизменяемы, она живёт до сборки мусора.
	privateBytes := []byte(privatePEM)
	defer clear(privateBytes)

	privateSealed, err := sealed.Seal(ctx, m.crypto, m.masterKeyHex, privateBytes)
	if err != nil {
		return nil, fmt.Errorf("keys: seal key %d: %w", userID, err)
	}

	return &ports.UserKey{
		UserID:        userID,
		Bits:          m.bits,
		PublicKeyPEM:  publicPEM,
		PrivateSealed: privateSealed,
		KeygenMs:      keygenMs,
		Status:        ports.KeyStatusReady,
	}, nil
}

// markFailed записывает статус failed, чтобы следующий EnsureUserKey
// запустил генерацию заново.
func (m *Manager) markFailed(parent context.Context, userID int64) {
	ctx, cancel := context.WithTimeout(parent, failedWriteTimeout)
	defer cancel()

	err := m.store.SaveUserKey(ctx, &ports.UserKey{UserID: userID, Bits: m.bits, Status: ports.KeyStatusFailed})
	if err != nil {
		m.log.ErrorContext(ctx, "save failed key status", "user_id", userID, "err", err)
	}
}

func (m *Manager) finish(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.running, userID)
}
