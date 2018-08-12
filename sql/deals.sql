DROP TABLE IF EXISTS deals, deal_categories, deal_memberships, deal_images, deal_votes, deal_comments CASCADE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- uuid
CREATE EXTENSION IF NOT EXISTS "postgis";    -- geography & location
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- similarity

CREATE TABLE deals
(
  id              uuid primary key default uuid_generate_v4(),
  title           text not null,
  description     text not null,
  thumbnail_id    uuid,
  latitude        float,
  longitude       float,
  point           geography,
  location_text   text,
  total_price     decimal(15,2),
  total_savings   decimal(15,2),
  quantity        int,
  category_id     serial not null,
  poster_id       uuid not null,
  posted_at       timestamp default now(),
  updated_at      timestamp,
  inactive_at     timestamp,
  city_id         serial references cities(id),
  CHECK (length(title) <= 128),
  CHECK (length(description) <= 512),
  CHECK (length(location_text) <= 128)
);

CREATE TABLE deal_categories
(
  id              smallserial primary key,
  name            text not null,
  max_images      integer default 12,
  max_active_days integer default 21,
  CHECK (length(name) <= 42)
);

CREATE TABLE deal_memberships
(
  id          uuid primary key default uuid_generate_v4(),
  user_id     uuid references users(id),
  deal_id     uuid references deals(id),
  joined_at   timestamp default now(),
  left_at     timestamp,
  UNIQUE(user_id, deal_id)
);

CREATE TABLE deal_images
(
  id          uuid primary key default uuid_generate_v4(),
  deal_id     uuid references deals(id),
  image_url   text not null,
  poster_id   uuid references users(id),
  posted_at   timestamp default now(),
  removed_at  timestamp,
  CHECK (length(image_url) <= 256) -- refer to cloudinary public id max len
);

CREATE TABLE deal_likes
(
  id          uuid primary key default uuid_generate_v4(),
  deal_id     uuid references deals(id),
  user_id     uuid references users(id),
  posted_at   timestamp default now(),
  is_upvote   bool,
  -- nullable for no vote
  UNIQUE(user_id, deal_id)
);

CREATE TABLE deal_comments
(
  id          uuid primary key default uuid_generate_v4(),
  deal_id     uuid references deals(id),
  user_id     uuid references users(id),
  comment_str text not null,
  posted_at   timestamp default now(),
  removed_at  timestamp,
  CHECK (length(comment_str) <= 256)
);


ALTER TABLE deals ADD CONSTRAINT deals_thumbnail_id_fkey
FOREIGN KEY (thumbnail_id) REFERENCES deal_images(id) ON DELETE CASCADE;
ALTER TABLE deals ADD CONSTRAINT deals_category_id_fkey
FOREIGN KEY (category_id) REFERENCES deal_categories(id) ON DELETE CASCADE;
ALTER TABLE deals ADD CONSTRAINT deals_poster_id_fkey
FOREIGN KEY (poster_id) REFERENCES users(id) ON DELETE CASCADE;


BEGIN;
DO $$
DECLARE
  vDealId uuid := 'f3c80460-de56-42c4-855f-82dda631fee1';
  vThumbId uuid := '93dda1a7-67a4-4e81-abcf-f3a2aba687f4';
  vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';
  vImageUrl text := 'https://via.placeholder.com/350x150.jpg';
  vCatId int := 1;
  vCityId int := 37541;
  vLat decimal := 1.3501484;
  vLong decimal := 103.8486871;
BEGIN
  INSERT INTO deal_categories (name) VALUES
    ('shirts'), ('pants'), ('food'), ('sneakers');

  INSERT INTO deal_images (id, image_url, poster_id) VALUES (
    vThumbId, vImageUrl , vUserId
  );

  INSERT INTO deals (
    id,
    title, description,
    latitude, longitude, point,
    location_text, total_price, total_savings, quantity,
    category_id, poster_id, city_id)
  VALUES (
    vDealId,
    'deal1', 'some shirt',
    vLat, vLong, ST_MakePoint(103.8198, 1.3521),
    'singapura mall', 40, 10.5, 2,
    vCatId, vUserId, vCityId
  );

  INSERT INTO deal_memberships (user_id, deal_id) VALUES (
    vUserId, vDealId
  );


END
$$;
COMMIT;
