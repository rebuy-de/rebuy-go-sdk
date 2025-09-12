-- Repeatable migration: Demo data for the full example application
-- This migration uses upserts to safely insert or update demo data

-- Insert or update sample users using upserts
insert into full_example.users (name, email, created_at) values
    ('Alice Smith', 'alice@example.com', now() - interval '72 hours'),
    ('Bob Johnson', 'bob@example.com', now() - interval '48 hours'), 
    ('Carol Williams', 'carol@example.com', now() - interval '24 hours')
on conflict (email) 
do update set 
    name = excluded.name,
    updated_at = now();

-- Insert welcome posts for each user (skip if already exists)
insert into full_example.posts (user_id, title, content, published, created_at)
select 
    u.id,
    'Welcome to ' || u.name || '''s Blog',
    'This is the first post by ' || u.name || '. Welcome to our platform!',
    true,
    u.created_at + interval '1 hour'
from full_example.users u
left join full_example.posts p on p.user_id = u.id and p.title = 'Welcome to ' || u.name || '''s Blog'
where p.id is null;

-- Add Alice's draft post (skip if already exists)
insert into full_example.posts (user_id, title, content, published, created_at)
select
    u.id,
    'Draft Post by ' || u.name,
    'This is a draft post that is not yet published. It contains some interesting ideas that are still being developed.',
    false,
    u.created_at + interval '2 hours'
from full_example.users u
left join full_example.posts p on p.user_id = u.id and p.title = 'Draft Post by ' || u.name
where u.email = 'alice@example.com' and p.id is null;
