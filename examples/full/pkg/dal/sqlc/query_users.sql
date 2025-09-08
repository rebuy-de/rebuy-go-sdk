-- User management queries

-- name: GetUserByID :one
select * from full_example.users
where id = $1 limit 1;

-- name: GetUserByEmail :one
select * from full_example.users
where email = $1 limit 1;

-- name: ListUsers :many
select * from full_example.users
order by created_at desc;

-- name: ListUsersWithPagination :many
select * from full_example.users
order by created_at desc
limit $1 offset $2;

-- name: CreateUser :one
insert into full_example.users (
    name, email
) values (
    $1, $2
)
returning *;

-- name: UpdateUser :one
update full_example.users
set 
    name = $2,
    email = $3,
    updated_at = now()
where id = $1
returning *;

-- name: DeleteUser :exec
delete from full_example.users
where id = $1;

-- name: CountUsers :one
select count(*) from full_example.users;
