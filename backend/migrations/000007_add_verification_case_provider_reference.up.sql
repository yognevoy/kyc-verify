ALTER TABLE verification_cases ADD COLUMN provider_reference TEXT;

CREATE UNIQUE INDEX idx_verification_cases_provider_reference
    ON verification_cases (provider_reference)
    WHERE provider_reference IS NOT NULL;
