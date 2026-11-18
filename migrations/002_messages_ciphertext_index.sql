-- Индекс для поиска сообщения по шифртексту: расшифрование Кузнечиком
-- находит по нему IV, которого нет в запросе фронта.
-- hash, а не btree: шифртекст RSA-32768 — 4 КБ, больше предела строки btree.

CREATE INDEX IF NOT EXISTS messages_ciphertext_idx ON messages USING hash (ciphertext);
