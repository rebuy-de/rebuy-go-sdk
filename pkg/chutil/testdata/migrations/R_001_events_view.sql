CREATE OR REPLACE VIEW events_by_name AS
SELECT name, count() AS total
FROM events
GROUP BY name
