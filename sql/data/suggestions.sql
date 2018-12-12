BEGIN;
DO $$
DECLARE
  vSuggestionId uuid := '6ba7b810-9dad-11d1-80b4-00c04fd430c8';
  vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';  -- same as in user-deals.sql
  vCatId int := 1;
  vTestImage text := 'https://via.placeholder.com/300x100?text=Visit+Blogging.com+Now';
  vActiveFrom timestamp := now() AT TIME ZONE 'utc';
  vInactiveBy timestamp := '2019-01-01';

BEGIN
  INSERT INTO suggestions (id, search_string, poster_id, category_id, active_from, inactive_by, banner_url)
  VALUES (vSuggestionId, 'test suggestion', vUserId, vCatId, vActiveFrom, vInactiveBy, vTestImage);
  INSERT INTO suggestions (poster_id, category_id, active_from, inactive_by)
  VALUES (vUserId, vCatId, vActiveFrom, vInactiveBy);
END
$$;
COMMIT;