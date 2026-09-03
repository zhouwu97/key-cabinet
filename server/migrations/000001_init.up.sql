-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Users table
CREATE TABLE users (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    student_id VARCHAR(50),
    phone VARCHAR(20),
    role VARCHAR(20) NOT NULL DEFAULT 'STUDENT',
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_student_id ON users(student_id);
CREATE INDEX idx_users_status ON users(status);

-- User identities table
CREATE TABLE user_identities (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    identity_type VARCHAR(20) NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(identity_type, provider_user_id)
);

CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);

-- Devices table
CREATE TABLE devices (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    location VARCHAR(200),
    capacity INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'ONLINE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Slots table
CREATE TABLE slots (
    id VARCHAR(64) PRIMARY KEY,
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    slot_number INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'EMPTY',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(device_id, slot_number)
);

CREATE INDEX idx_slots_device_id ON slots(device_id);
CREATE INDEX idx_slots_status ON slots(status);

-- Keys table
CREATE TABLE keys (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    key_number VARCHAR(50) NOT NULL UNIQUE,
    rfid_tag VARCHAR(100),
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id),
    slot_id VARCHAR(64) REFERENCES slots(id),
    category VARCHAR(50),
    status VARCHAR(20) NOT NULL DEFAULT 'AVAILABLE',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_keys_device_id ON keys(device_id);
CREATE INDEX idx_keys_status ON keys(status);
CREATE INDEX idx_keys_rfid_tag ON keys(rfid_tag);

-- Reservations table
CREATE TABLE reservations (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL REFERENCES users(id),
    key_id VARCHAR(64) NOT NULL REFERENCES keys(id),
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    purpose TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reservations_user_id ON reservations(user_id);
CREATE INDEX idx_reservations_key_id ON reservations(key_id);
CREATE INDEX idx_reservations_status ON reservations(status);
CREATE INDEX idx_reservations_time ON reservations(start_time, end_time);

-- Borrow records table
CREATE TABLE borrow_records (
    id VARCHAR(64) PRIMARY KEY,
    reservation_id VARCHAR(64) NOT NULL REFERENCES reservations(id),
    user_id VARCHAR(64) NOT NULL REFERENCES users(id),
    key_id VARCHAR(64) NOT NULL REFERENCES keys(id),
    status VARCHAR(20) NOT NULL DEFAULT 'BORROWED',
    borrowed_at TIMESTAMP NOT NULL,
    expected_return_at TIMESTAMP NOT NULL,
    returned_at TIMESTAMP,
    rfid_verified BOOLEAN DEFAULT FALSE,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_borrow_records_reservation_id ON borrow_records(reservation_id);
CREATE INDEX idx_borrow_records_user_id ON borrow_records(user_id);
CREATE INDEX idx_borrow_records_key_id ON borrow_records(key_id);
CREATE INDEX idx_borrow_records_status ON borrow_records(status);

-- Device operations table
CREATE TABLE device_operations (
    id VARCHAR(64) PRIMARY KEY,
    reservation_id VARCHAR(64) REFERENCES reservations(id),
    borrow_record_id VARCHAR(64) REFERENCES borrow_records(id),
    device_id VARCHAR(64) NOT NULL REFERENCES devices(id),
    slot_id VARCHAR(64) REFERENCES slots(id),
    key_id VARCHAR(64) REFERENCES keys(id),
    operation_type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'CREATED',
    initiated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_device_operations_reservation_id ON device_operations(reservation_id);
CREATE INDEX idx_device_operations_device_id ON device_operations(device_id);
CREATE INDEX idx_device_operations_status ON device_operations(status);
CREATE INDEX idx_device_operations_type ON device_operations(operation_type);
