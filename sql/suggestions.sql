ALTER TABLE suggestions
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
  inactive_by       timestamp,
  CHECK (length(search_string) <= 128),
  CHECK (length(banner_url) <= 256)
);

ALTER TABLE suggestions
  ADD CONSTRAINT suggestions_poster_id_fkey FOREIGN KEY (poster_id) REFERENCES users(id) ON DELETE CASCADE,
  ADD CONSTRAINT suggestions_deal_category_id_fkey FOREIGN KEY (category_id) REFERENCES deal_categories(id) ON DELETE CASCADE;

BEGIN;
DO $$
DECLARE
  vSuggestionId uuid := '6ba7b810-9dad-11d1-80b4-00c04fd430c8';
  vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';  -- same as in deals.sql
  vCatId int := 1;
  vTestImage text := 'https://via.placeholder.com/300x100?text=Visit+Blogging.com+Now';
  vInactiveBy timestamp := '2019-01-01';

BEGIN
  INSERT INTO suggestions (id, search_string, poster_id, category_id, inactive_by, banner_url)
  VALUES (vSuggestionId, 'test suggestion', vUserId, vCatId, vInactiveBy, vTestImage);
  INSERT INTO suggestions (poster_id, category_id, inactive_by)
  VALUES (vUserId, vCatId, vInactiveBy);
END
$$;
COMMIT;