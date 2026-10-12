CREATE TABLE IF NOT EXISTS contact_offers (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    request_id    UUID         NOT NULL,
    offerer_phone VARCHAR(20)  NOT NULL,
    offerer_name  VARCHAR(100) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(request_id, offerer_phone)
);

CREATE INDEX IF NOT EXISTS idx_contact_offers_request_id    ON contact_offers(request_id);
CREATE INDEX IF NOT EXISTS idx_contact_offers_offerer_phone ON contact_offers(offerer_phone);
