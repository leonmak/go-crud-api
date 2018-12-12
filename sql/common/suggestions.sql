ALTER TABLE IF EXISTS suggestions
  DROP CONSTRAINT IF EXISTS suggestions_poster_id_fkey,
  DROP CONSTRAINT IF EXISTS suggestions_deal_category_id_fkey;

DROP TABLE IF EXISTS suggestions CASCADE;
CREATE TABLE suggestions
(
  id                uuid primary key default uuid_generate_v4(),
  search_string     text,
  poster_id         uuid,
  category_id       smallserial,
  latitude          float,
  longitude         float,
  radius_km         float,
  banner_url        text,
  active_from       timestamp,
  inactive_by       timestamp,
  CHECK (length(search_string) <= 128),
  CHECK (length(banner_url) <= 256)
);

ALTER TABLE suggestions
  ADD CONSTRAINT suggestions_poster_id_fkey FOREIGN KEY (poster_id) REFERENCES users(id) ON DELETE CASCADE,
  ADD CONSTRAINT suggestions_deal_category_id_fkey FOREIGN KEY (category_id) REFERENCES deal_categories(id) ON DELETE CASCADE;
