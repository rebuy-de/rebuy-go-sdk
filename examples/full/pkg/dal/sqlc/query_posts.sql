-- Post management queries

-- name: GetPostByID :one
select * from full_example.posts
where id = $1 limit 1;

-- name: ListPosts :many
select * from full_example.posts
order by created_at desc;

-- name: ListPublishedPosts :many
select * from full_example.posts
where published = true
order by created_at desc;

-- name: ListPostsByUser :many
select * from full_example.posts
where user_id = $1
order by created_at desc;

-- name: ListPostsWithUsers :many
select 
    p.id,
    p.title,
    p.content,
    p.published,
    p.created_at,
    p.updated_at,
    u.name as author_name,
    u.email as author_email
from full_example.posts p
join full_example.users u on u.id = p.user_id
where p.published = true
order by p.created_at desc;

-- name: CreatePost :one
insert into full_example.posts (
    user_id, title, content, published
) values (
    $1, $2, $3, $4
)
returning *;

-- name: UpdatePost :one
update full_example.posts
set 
    title = $2,
    content = $3,
    published = $4,
    updated_at = now()
where id = $1
returning *;

-- name: PublishPost :one
update full_example.posts
set 
    published = true,
    updated_at = now()
where id = $1
returning *;

-- name: DeletePost :exec
delete from full_example.posts
where id = $1;
