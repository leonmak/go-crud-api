-- DROP PREVIOUS TABLE
ALTER TABLE IF EXISTS deals
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
  icon_url        text,
  priority        int default 1, -- bigger is more important
  is_active       boolean default true,
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
END

$$; -- end DO
COMMIT; -- end BEGIN
