BEGIN;

DO $$
DECLARE
  vUserId uuid := 'eab30e15-fded-46fc-93f4-af0cb2a0ebd8';
BEGIN
  INSERT INTO users (id, email, display_name, country_code)
  VALUES (vUserId, 'leon.mak@u.nus.edu', 'leon', 'US');
END
$$;

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
  vUserId, vDealId);
END

$$; -- end DO
COMMIT; -- end BEGIN
