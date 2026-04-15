CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    phone VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(100),
    role VARCHAR(10) NOT NULL CHECK (role IN ('passenger', 'driver')),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE trips (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    passenger_id UUID REFERENCES users(id),
    driver_id UUID REFERENCES users(id),
    from_lat DOUBLE PRECISION NOT NULL,
    from_lng DOUBLE PRECISION NOT NULL,
    to_lat DOUBLE PRECISION NOT NULL,
    to_lng DOUBLE PRECISION NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    price INT DEFAULT 500,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE drivers (
    id UUID PRIMARY KEY REFERENCES users(id),
    is_online BOOLEAN DEFAULT FALSE,
    current_lat DOUBLE PRECISION,
    current_lng DOUBLE PRECISION,
    updated_at TIMESTAMP DEFAULT NOW()
);