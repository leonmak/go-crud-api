-- DROP PREVIOUS TABLE
ALTER TABLE deals
  DROP CONSTRAINT IF EXISTS deals_thumbnail_id_fkey,
  DROP CONSTRAINT IF EXISTS deals_category_id_fkey,
  DROP CONSTRAINT IF EXISTS deals_poster_id_fkey;
DROP TABLE IF EXISTS deals, deal_categories, deal_likes, deal_memberships, deal_images, deal_comments CASCADE;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";  -- uuid
CREATE EXTENSION IF NOT EXISTS "postgis";    -- geography & location
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- similarity


-- CREATE NEW TABLES
CREATE TABLE deals
(
  id                uuid primary key default uuid_generate_v4(),
  title             text not null,
  description       text not null,
  category_id       serial not null,
  total_price       decimal(15,2),
  quantity          int,
  benefits          text,
  thumbnail_id      uuid,
  latitude          float,
  longitude         float,
  point             geography,
  location_text     text,
  poster_id         uuid not null,
  posted_at         timestamp default timezone('utc', now()),
  updated_at        timestamp,
  inactive_at       timestamp,
  is_featured       boolean default false,
  featured_url      text,
  country_code      char(2),
  CHECK (length(title) <= 128),
  CHECK (length(benefits) <= 128),
  CHECK (length(description) <= 512),
  CHECK (length(location_text) <= 128),
  CHECK (length(featured_url) <= 2048)
);

CREATE TABLE deal_categories
(
  id              smallserial primary key,
  name            text not null,
  display_name    text not null,
  CHECK (length(name) <= 42),
  CHECK (length(display_name) <= 42)
);

CREATE TABLE deal_memberships
(
  id          uuid primary key default uuid_generate_v4(),
  user_id     uuid references users(id),
  deal_id     uuid references deals(id),
  joined_at   timestamp default timezone('utc', now()),
  UNIQUE(user_id, deal_id)
);

CREATE TABLE deal_images
(
  id          uuid primary key default uuid_generate_v4(),
  deal_id     uuid references deals(id),
  image_url   text not null,
  poster_id   uuid references users(id),
  posted_at   timestamp default timezone('utc', now()),
  removed_at  timestamp,
  CHECK (length(image_url) <= 256) -- refer to cloudinary public id max len
);

CREATE TABLE deal_likes
(
  id          uuid primary key default uuid_generate_v4(),
  deal_id     uuid references deals(id),
  user_id     uuid references users(id),
  posted_at   timestamp default timezone('utc', now()),
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
  posted_at   timestamp default timezone('utc', now()),
  removed_at  timestamp,
  CHECK (length(comment_str) <= 256)
);

ALTER TABLE deals
  ADD CONSTRAINT deals_thumbnail_id_fkey FOREIGN KEY (thumbnail_id) REFERENCES deal_images(id) ON DELETE CASCADE,
  ADD CONSTRAINT deals_category_id_fkey FOREIGN KEY (category_id) REFERENCES deal_categories(id) ON DELETE CASCADE,
  ADD CONSTRAINT deals_poster_id_fkey FOREIGN KEY (poster_id) REFERENCES users(id) ON DELETE CASCADE;


-- FILL INITIAL TABLE
BEGIN;

DO $$
DECLARE
  vDealId uuid := 'f3c80460-de56-42c4-855f-82dda631fee1';
  vDealId2 uuid := 'f3c80460-de56-42c4-855f-82dda631fee2';
  vThumbId uuid := '93dda1a7-67a4-4e81-abcf-f3a2aba687f4';
  vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';
  vImageUrl text := 'https://via.placeholder.com/350x100.jpg';
  vCatId int := 1;
  vLat float := 1.3501484;
  vLong float := 103.8486871;
  vImageId uuid;

BEGIN
  INSERT INTO deal_categories (name, display_name) VALUES
    ('app', 'Apps'),
    ('concert', 'Concert'),
    ('gadgets', 'Gadgets'),
    ('games', 'Games'),
    ('men', 'Men''s'),
    ('arts', 'Arts'),
    ('cycling', 'Sports'),
    ('eyewear', 'Eyewear'),
    ('gift', 'Gifts'),
    ('movie', 'Movies'),
    ('snacks', 'Snacks'),
    ('book', 'Books'),
    ('finance', 'Finance'),
    ('fast-food', 'Fast Food'),
    ('plane', 'Plane'),
    ('takeaway', 'Takeout'),
    ('cafe', 'Cafe'),
    ('drinks', 'Drinks'),
    ('footwear', 'Footwear'),
    ('karaoke', 'Karaoke'),
    ('sale', 'Sales'),
    ('women', 'Women''s');

  INSERT INTO deal_images (id, image_url, poster_id) VALUES (
    vThumbId, vImageUrl , vUserId
  ) RETURNING id INTO vImageId;

  INSERT INTO deals (id, title, description, thumbnail_id,
                     latitude, longitude, point, country_code,
                     location_text, total_price, benefits, quantity,
                     category_id, poster_id, posted_at, is_featured)
  VALUES (vDealId, 'deal1', 'some shirt', vImageId,
          vLat, vLong, ST_MakePoint(103.8198, 1.3521), 'US',
          'p. singapura mall', 40, 'get 10% cashback', 2,
          vCatId, vUserId, now() AT TIME ZONE 'UTC', false);

  INSERT INTO deals (id, title, description, thumbnail_id,
                     latitude, longitude, point, country_code,
                     location_text, total_price, benefits, quantity,
                     category_id, poster_id, posted_at, is_featured)
  VALUES (vDealId2, 'deal2', 'some official shirt', vImageId,
          vLat, vLong, ST_MakePoint(103.8198, 1.3521), 'US',
         'p. singapura mall', 40, 'get 10% cashback', 2,
          vCatId, vUserId, now() AT TIME ZONE 'UTC', true);

  INSERT INTO deal_memberships (user_id, deal_id) VALUES (
    vUserId, vDealId
  );
END

$$; -- end DO
COMMIT; -- end BEGIN
