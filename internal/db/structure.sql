DO ' BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = ''order_status_type'') THEN
        CREATE TYPE order_status_type AS ENUM (''NEW'', ''PROCESSING'', ''INVALID'', ''PROCESSED'');
    END IF;
END ';

CREATE TABLE IF NOT EXISTS users
(
    uuid       UUID                     DEFAULT gen_random_uuid(),
    login      VARCHAR(255) NOT NULL,
    password   VARCHAR(60) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (uuid),
    CONSTRAINT login_uidx UNIQUE (login)
);

CREATE TABLE IF NOT EXISTS balance
(
    id SERIAL,
    user_uuid UUID,
    current BIGINT DEFAULT 0,
    withdrawn BIGINT DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT user_uuid_uidx UNIQUE (user_uuid)
);

CREATE TABLE IF NOT EXISTS withdrawals
(
    uuid UUID DEFAULT gen_random_uuid(),
    user_uuid UUID,
    order_id VARCHAR(20) NOT NULL,
    sum BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (uuid)
);

CREATE INDEX IF NOT EXISTS user_uuid_idx ON withdrawals (user_uuid);

CREATE TABLE IF NOT EXISTS orders
(
    id SERIAL,
    user_uuid UUID,
    order_id VARCHAR(20) NOT NULL,
    status order_status_type NOT NULL,
    accrual BIGINT DEFAULT NULL,
    uploaded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (id),
    CONSTRAINT order_id_uidx UNIQUE (order_id)
);