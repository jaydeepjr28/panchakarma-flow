-- PostgreSQL Multi-Database Initialization Script
-- Automatically executed on first boot by Docker entrypoint

CREATE DATABASE identity_db;
CREATE DATABASE clinic_db;
CREATE DATABASE appointment_db;
CREATE DATABASE therapy_db;
CREATE DATABASE notification_db;
CREATE DATABASE feedback_analytics_db;

-- Grant permissions to default panchakarma user
GRANT ALL PRIVILEGES ON DATABASE identity_db TO panchakarma;
GRANT ALL PRIVILEGES ON DATABASE clinic_db TO panchakarma;
GRANT ALL PRIVILEGES ON DATABASE appointment_db TO panchakarma;
GRANT ALL PRIVILEGES ON DATABASE therapy_db TO panchakarma;
GRANT ALL PRIVILEGES ON DATABASE notification_db TO panchakarma;
GRANT ALL PRIVILEGES ON DATABASE feedback_analytics_db TO panchakarma;

-- Connect to each database and create required extensions
\c identity_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c clinic_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\c appointment_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

\c therapy_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c notification_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c feedback_analytics_db
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
