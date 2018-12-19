DROP TABLE IF EXISTS users CASCADE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE users
(
  -- uuid for dynamic tables for easier sharding
  id                    uuid primary key default uuid_generate_v4(),
  email                 citext not null unique,
  display_name          text not null,
  image_url             text,
  country_code          char(2),
  auth_type             text,
  fir_id                text,
  created_at            timestamp default timezone('utc', now()),
  CHECK (length(display_name) <= 42),
  CHECK (length(image_url) <= 256)
);

CREATE UNIQUE INDEX users_unique_lower_email_idx ON users (lower(email));

