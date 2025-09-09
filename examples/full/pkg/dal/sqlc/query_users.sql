-- User management queries

-- name: ListUsers :many
select * from full_example.users
order by created_at desc;
