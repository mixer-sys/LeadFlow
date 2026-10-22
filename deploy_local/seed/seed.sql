BEGIN;

-- organizations
INSERT INTO organizations (name, telegram_bot_token, telegram_chat_id)
SELECT 'LeadFlow', '', 0
WHERE NOT EXISTS (
    SELECT 1 FROM organizations WHERE name = 'LeadFlow'
);

-- leads
INSERT INTO leads (organization_id, source, name, email, phone, message, status)
SELECT o.id, 'manual', 'Иван Петров', 'ivan@example.com', '+79990000001', 'Тестовая заявка', 'new'
FROM organizations o
WHERE o.name = 'LeadFlow'
  AND NOT EXISTS (
      SELECT 1 FROM leads l WHERE l.email = 'ivan@example.com'
  );

INSERT INTO leads (organization_id, source, name, email, phone, message, status)
SELECT o.id, 'manual', 'Мария Смирнова', 'maria@example.com', '+79990000002', 'Тестовая заявка', 'new'
FROM organizations o
WHERE o.name = 'LeadFlow'
  AND NOT EXISTS (
      SELECT 1 FROM leads l WHERE l.email = 'maria@example.com'
  );

-- api_keys
INSERT INTO api_keys (organization_id, key, name, is_active)
SELECT o.id, 'local-test-api-key', 'local-test-key', true
FROM organizations o
WHERE o.name = 'LeadFlow'
ON CONFLICT (key) DO NOTHING;

-- webhooks
INSERT INTO webhooks (url, is_active)
SELECT 'https://example.com/webhook', true
WHERE NOT EXISTS (
    SELECT 1 FROM webhooks WHERE url = 'https://example.com/webhook'
);

COMMIT;
