drop table if exists users;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

create table users (
  id              uuid primary key default uuid_generate_v4(),
  email           citext not null unique,
  password_digest text not null,
  display_name    varchar(36) not null,
  image_url       text,
  verify_attempt  date,
  verified        bool default false
);

create unique index users_unique_lower_email_idx on users (lower(email));

insert into users (email, password_digest, display_name)
  values ('leon.mak@u.nus.edu', uuid_generate_v4(),'leon');
