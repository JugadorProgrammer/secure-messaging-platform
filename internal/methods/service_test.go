package methods

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// opRecord — запись, ушедшая в журнал операций.
type opRecord struct {
	userID *int64
	op     string
	ref    string
	ok     bool
}

// fakeStore — хранилище в памяти. Нереализованные методы ports.Store не
// участвуют в тестах этого пакета: обращение к ним уронит тест паникой,
// а не тихо пройдёт.
type fakeStore struct {
	ports.Store

	rows     []ports.MethodRow
	listErr  error
	setErr   error
	logErr   error
	setCalls []string
	ops      []opRecord
}

func newFakeStore(rows ...ports.MethodRow) *fakeStore {
	return &fakeStore{rows: rows}
}

func (f *fakeStore) Methods(context.Context) ([]ports.MethodRow, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}

	return slices.Clone(f.rows), nil
}

func (f *fakeStore) SetMethodEnabled(_ context.Context, code string, on bool) error {
	f.setCalls = append(f.setCalls, code)

	if f.setErr != nil {
		return f.setErr
	}

	for i := range f.rows {
		if f.rows[i].Code == code {
			f.rows[i].Enabled = on

			return nil
		}
	}

	return ports.ErrNotFound
}

func (f *fakeStore) LogOp(_ context.Context, userID *int64, op, ref string, ok bool) error {
	f.ops = append(f.ops, opRecord{userID: userID, op: op, ref: ref, ok: ok})

	return f.logErr
}

func seedRows() []ports.MethodRow {
	return []ports.MethodRow{
		{Code: "rsa", Title: "RSA", KeySize: 2048, Enabled: true},
		{Code: "kuznyechik", Title: "Кузнечик", KeySize: 256, Enabled: true},
	}
}

func TestServiceListSortsByCode(t *testing.T) {
	svc := NewService(newFakeStore(seedRows()...))

	dtos, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(dtos) != 2 {
		t.Fatalf("len(dtos) = %d, want 2", len(dtos))
	}

	// kuznyechik идёт раньше rsa по алфавиту, независимо от порядка в хранилище.
	if dtos[0].Code != ports.MethodKuznyechik || dtos[1].Code != ports.MethodRSA {
		t.Errorf("codes = %v, %v, want kuznyechik, rsa", dtos[0].Code, dtos[1].Code)
	}
	if dtos[1].Title != "RSA" || dtos[1].KeySize != 2048 || !dtos[1].Enabled {
		t.Errorf("rsa row = %+v", dtos[1])
	}
}

func TestServiceListEmptyIsNotNil(t *testing.T) {
	svc := NewService(newFakeStore())

	dtos, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if dtos == nil {
		t.Fatal("List() = nil, want empty slice: в JSON должен попадать [] а не null")
	}
	if len(dtos) != 0 {
		t.Fatalf("len(dtos) = %d, want 0", len(dtos))
	}
}

func TestServiceListWrapsStoreError(t *testing.T) {
	storeErr := errors.New("connection reset")
	svc := NewService(&fakeStore{listErr: storeErr})

	if _, err := svc.List(context.Background()); !errors.Is(err, storeErr) {
		t.Fatalf("List() error = %v, want wrapped %v", err, storeErr)
	}
}

func TestServiceSetEnabled(t *testing.T) {
	st := newFakeStore(seedRows()...)
	svc := NewService(st)

	dto, err := svc.SetEnabled(context.Background(), "rsa", false)
	if err != nil {
		t.Fatalf("SetEnabled() error = %v", err)
	}

	if dto.Enabled {
		t.Error("dto.Enabled = true, want false")
	}
	if dto.Code != ports.MethodRSA || dto.KeySize != 2048 {
		t.Errorf("dto = %+v", dto)
	}
	if len(st.setCalls) != 1 || st.setCalls[0] != "rsa" {
		t.Errorf("setCalls = %v, want [rsa]", st.setCalls)
	}
}

func TestServiceSetEnabledUnknownCode(t *testing.T) {
	st := newFakeStore(seedRows()...)
	svc := NewService(st)

	// Неизвестный код отсекается до обращения к базе.
	if _, err := svc.SetEnabled(context.Background(), "gost", true); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("SetEnabled() error = %v, want ports.ErrNotFound", err)
	}

	if len(st.setCalls) != 0 {
		t.Errorf("setCalls = %v, want пустой: изменения не было", st.setCalls)
	}
}

func TestServiceSetEnabledStoreError(t *testing.T) {
	storeErr := errors.New("deadlock detected")
	svc := NewService(&fakeStore{setErr: storeErr})

	if _, err := svc.SetEnabled(context.Background(), "rsa", true); !errors.Is(err, storeErr) {
		t.Fatalf("SetEnabled() error = %v, want wrapped %v", err, storeErr)
	}
}

func TestServiceSetEnabledRowMissingAfterUpdate(t *testing.T) {
	// Обновление прошло, но к моменту чтения строки в списке нет: ответ
	// собрать не из чего, и это тот же 404, что и для неизвестного кода.
	svc := NewService(&raceStore{fakeStore: fakeStore{rows: seedRows()}})

	if _, err := svc.SetEnabled(context.Background(), "rsa", true); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("SetEnabled() error = %v, want ports.ErrNotFound", err)
	}
}

// raceStore имитирует гонку: метод обновлён, но к моменту чтения список пуст.
type raceStore struct {
	fakeStore
}

func (r *raceStore) Methods(ctx context.Context) ([]ports.MethodRow, error) {
	if len(r.setCalls) > 0 {
		return nil, nil
	}

	return r.fakeStore.Methods(ctx)
}

func TestIsKnownCode(t *testing.T) {
	cases := []struct {
		code string
		want bool
	}{
		{"rsa", true},
		{"kuznyechik", true},
		{"RSA", false},
		{"", false},
		{"gost", false},
		{"rsa ", false},
	}

	for _, tc := range cases {
		if got := IsKnownCode(tc.code); got != tc.want {
			t.Errorf("IsKnownCode(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestMethodEnabled(t *testing.T) {
	cases := []struct {
		name    string
		code    string
		enabled bool
		want    bool
	}{
		{"включён", "rsa", true, true},
		{"выключен", "rsa", false, false},
		{"неизвестный код", "gost", true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := []ports.MethodRow{{Code: "rsa", Title: "RSA", KeySize: 2048, Enabled: tc.enabled}}

			got, err := MethodEnabled(context.Background(), newFakeStore(rows...), tc.code)
			if err != nil {
				t.Fatalf("MethodEnabled() error = %v", err)
			}

			if got != tc.want {
				t.Errorf("MethodEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMethodEnabledStoreError(t *testing.T) {
	storeErr := errors.New("connection refused")

	got, err := MethodEnabled(context.Background(), &fakeStore{listErr: storeErr}, "rsa")
	if !errors.Is(err, storeErr) {
		t.Fatalf("MethodEnabled() error = %v, want wrapped %v", err, storeErr)
	}
	if got {
		t.Error("MethodEnabled() = true, want false при ошибке базы")
	}
}
