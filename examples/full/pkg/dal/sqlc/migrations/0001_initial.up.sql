-- Initial schema for the full example application

-- Create schema for the application
create schema if not exists full_example;

-- Users table - demonstrates basic CRUD operations
create table full_example.users (
    id uuid primary key default gen_random_uuid(),
    name varchar not null,
    email varchar not null unique,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Posts table - demonstrates relationships and more complex queries
create table full_example.posts (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references full_example.users(id) on delete cascade,
    title varchar not null,
    content text not null,
    published boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- Add indexes for better performance
create index users_email_idx on full_example.users(email);
create index users_created_at_idx on full_example.users(created_at);

create index posts_user_id_idx on full_example.posts(user_id);
create index posts_published_idx on full_example.posts(published);
create index posts_created_at_idx on full_example.posts(created_at);

-- Insert sample data for demonstration
insert into full_example.users (name, email, created_at) values
    ('Alice Smith', 'alice@example.com', now() - interval '72 hours'),
    ('Bob Johnson', 'bob@example.com', now() - interval '48 hours'), 
    ('Carol Williams', 'carol@example.com', now() - interval '24 hours');

-- Add some sample posts
insert into full_example.posts (user_id, title, content, published, created_at) 
select 
    u.id,
    'Welcome to ' || u.name || '''s Blog',
    'This is the first post by ' || u.name || '. Welcome to our platform!',
    true,
    u.created_at + interval '1 hour'
from full_example.users u;

insert into full_example.posts (user_id, title, content, published, created_at)
select
    u.id,
    'Draft Post by ' || u.name,
    'This is a draft post that is not yet published.',
    false,
    u.created_at + interval '2 hours'
from full_example.users u
where u.email = 'alice@example.com';
