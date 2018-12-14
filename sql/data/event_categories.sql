BEGIN;
INSERT INTO deal_categories (name, display_name, priority, icon_url)
    VALUES ('christmas', 'Christmas', 3, 'https://res.cloudinary.com/groupbuying/image/upload/v1544771570/event_icons/xmas-bag.png');
INSERT INTO deal_categories (name, display_name, priority, icon_url)
    VALUES ('cny', 'New Year', 2, 'https://res.cloudinary.com/groupbuying/image/upload/v1544773291/event_icons/hongbao.png');
END;