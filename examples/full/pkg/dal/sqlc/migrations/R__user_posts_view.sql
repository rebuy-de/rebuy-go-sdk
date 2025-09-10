-- Repeatable migration: Create user_posts view
-- This view joins users and posts tables for easier querying

create or replace view full_example.user_posts as
select 
    u.id as user_id,
    u.name as user_name,
    u.email as user_email,
    u.created_at as user_created_at,
    p.id as post_id,
    p.title as post_title,
    p.content as post_content,
    p.published as post_published,
    p.created_at as post_created_at,
    p.updated_at as post_updated_at,
    -- Computed fields for analytics
    case 
        when p.published then 'published'
        else 'draft'
    end as post_status,
    length(p.content) as content_length,
    extract(days from now() - p.created_at) as days_since_creation
from full_example.users u
left join full_example.posts p on u.id = p.user_id
order by u.name, p.created_at desc;
