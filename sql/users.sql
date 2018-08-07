DROP TABLE IF EXISTS users CASCADE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

CREATE TABLE users
(
  id                    uuid primary key default uuid_generate_v4(),
  email                 citext not null unique,
  display_name          text not null,
  password_digest       text not null,
  url_alias             text not null unique,
  image_url             text,
  verify_email_sent_at  timestamp,
  verified_at           timestamp,
  city_id               serial references cities(id),
  CHECK (length(display_name) <= 42),
  CHECK (length(password_digest) <= 60),
  CHECK (length(image_url) <= 255),
  CHECK (length(url_alias) <= 12)
);

CREATE UNIQUE INDEX users_unique_lower_email_idx ON users (lower(email));

INSERT INTO users (id, email, password_digest, display_name,
                   url_alias, city_id)
  VALUES ('93dda1a7-67a4-4e81-abcf-f3a2aba687f4', 'leon.mak@u.nus.edu', 'password_d','leon',
          'KFTGcuiQ9p', 37541);
