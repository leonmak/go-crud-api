DROP TABLE IF EXISTS users CASCADE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE users
(
  -- uuid for dynamic tables for easier sharding
  id                    uuid primary key default uuid_generate_v4(),
  email                 citext not null unique,
  display_name          text not null,
  password_digest       text,
  image_url             text,
  verify_email_sent_at  timestamp,
  verified_at           timestamp,
  city_id               serial references cities(id),
  CHECK (length(display_name) <= 42),
  CHECK (length(password_digest) <= 60),
  CHECK (length(image_url) <= 256)
);

CREATE UNIQUE INDEX users_unique_lower_email_idx ON users (lower(email));

BEGIN;
DO $$
  DECLARE
    vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';
  BEGIN
    INSERT INTO users (id, email, password_digest, display_name, city_id)
    VALUES (vUserId, 'leon.mak@u.nus.edu', 'password_d','leon', 37541);
  END
$$;
COMMIT;

