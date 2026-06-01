-- Widen phone columns to hold AES-256-GCM encrypted values (base64 ~80 chars).
ALTER TABLE rides         ALTER COLUMN phone          TYPE VARCHAR(100);
ALTER TABLE requests      ALTER COLUMN phone          TYPE VARCHAR(100);
ALTER TABLE interests     ALTER COLUMN searcher_phone TYPE VARCHAR(100);
ALTER TABLE subscriptions ALTER COLUMN phone          TYPE VARCHAR(100);
