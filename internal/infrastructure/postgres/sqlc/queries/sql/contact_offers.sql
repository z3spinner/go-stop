-- name: InsertContactOffer :exec
INSERT INTO contact_offers (id, request_id, offerer_phone, offerer_name)
VALUES ($1, $2, $3, $4)
ON CONFLICT (request_id, offerer_phone) DO NOTHING;

-- name: GetContactOfferByRequestAndOfferer :one
SELECT id, request_id, offerer_phone, offerer_name, created_at
FROM contact_offers WHERE request_id = $1 AND offerer_phone = $2;

-- name: ListContactOffersByRequest :many
SELECT id, request_id, offerer_phone, offerer_name, created_at
FROM contact_offers WHERE request_id = $1
ORDER BY created_at ASC;
