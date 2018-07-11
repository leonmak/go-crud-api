DROP TABLE IF EXISTS users;

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
  verify_email_sent_at  date,
  is_verified           bool default false,
  CHECK (length(display_name) <= 42),
  CHECK (length(password_digest) <= 60),
  CHECK (length(image_url) <= 255),
  CHECK (length(url_alias) <= 12)
);

CREATE UNIQUE INDEX users_unique_lower_email_idx ON users (lower(email));

INSERT INTO users (email, password_digest, display_name, url_alias)
  VALUES ('leon.mak@u.nus.edu', '93dda1a7-67a4-4e81-abcf-f3a2aba687f4','leon', 'KFTGcuiQ9p');
