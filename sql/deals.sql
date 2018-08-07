DROP TABLE IF EXISTS deals, deal_categories, deal_memberships, deal_images, deal_votes, deal_comments;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- uuid
CREATE EXTENSION IF NOT EXISTS "postgis";    -- geography & location
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- similarity

CREATE TABLE deal_categories
(
  id              smallserial primary key,
  name            text not null,
  max_images      integer default 12,
  max_active_days integer default 21,
  CHECK (length(name) <= 42)
);

INSERT INTO deal_categories (name) VALUES ('shirts');

CREATE TABLE deals
(
  id              uuid primary key default uuid_generate_v4(),
  title           text not null,
  description     text not null,
  url_alias       text not null unique,
  latitude        float,
  longitude       float,
  point           geography,
  location_text   text,
  expected_price  serial,
  category_id     serial references deal_categories(id),
  poster_id       uuid references users(id),
  posted_at       timestamp default now(),
  updated_at      timestamp,
  inactive_at     timestamp,
  city_id         serial references cities(id),
  CHECK (length(title) <= 128),
  CHECK (length(description) <= 512),
  CHECK (length(url_alias) <= 12),
  CHECK (length(location_text) <= 128)
);

INSERT INTO deals (
  id, title, description, url_alias,
  latitude, longitude, point,
  location_text, expected_price,
  category_id, poster_id, city_id)
VALUES (
  uuid_generate_v4(), 'deal1', 'some shirt', 'uadsfa324D',
  1.3521, 103.8198, ST_MakePoint(103.8198, 1.3521),
  'singapura mall', 1.4,
  1, '93dda1a7-67a4-4e81-abcf-f3a2aba687f4', 37541);

CREATE TABLE deal_memberships
(
  id          bigserial primary key,
  user_id     uuid references users(id),
  deal_id     uuid references deals(id),
  joined_at   timestamp default now(),
  left_at     timestamp
);

CREATE TABLE deal_images
(
  id          bigserial primary key,
  deal_id     uuid references deals(id),
  image_url   text,
  poster_id   uuid references users(id),
  posted_at   timestamp default now(),
  CHECK (length(image_url) <= 255)
);

CREATE TABLE deal_votes
(
  id          bigserial primary key,
  deal_id     uuid references deals(id),
  user_id     uuid references users(id),
  posted_at   timestamp default now()
);

CREATE TABLE deal_comments
(
  id          bigserial primary key,
  deal_id     uuid references deals(id),
  user_id     uuid references users(id),
  comment     text not null,
  posted_at   timestamp default now(),
  CHECK (length(comment) <= 255)
);
