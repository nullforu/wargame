-- Dummy seed data for local/dev testing (PostgreSQL)
-- Usage:
--   psql "$DATABASE_URL" -f scripts/sql/seed_dummy_data.sql
-- Test login password for all seeded users:
--   pass1234!

BEGIN;

TRUNCATE TABLE challenge_votes, submissions, stacks, challenges, users, affiliations RESTART IDENTITY CASCADE;

INSERT INTO affiliations (id, name, created_at, updated_at)
VALUES
    (1, '세명컴퓨터고등학교', NOW() - INTERVAL '100 days', NOW()),
    (2, '널포유', NOW() - INTERVAL '95 days', NOW()),
    (3, '삼셩전자', NOW() - INTERVAL '90 days', NOW()),
    (4, '서울에있는모대학교', NOW() - INTERVAL '85 days', NOW());

-- bcrypt("pass1234!", cost=10)
-- generated value: $2a$10$bsyMO/LWwVSIFN.LS09qbuPjVxwIvOqC3i79lJ6hzHw722cwLRa4m

INSERT INTO users (id, email, username, password_hash, role, affiliation_id, blocked_reason, blocked_at, created_at, updated_at)
VALUES
    (1, 'admin@wargame.local', 'admin', '$2a$10$bsyMO/LWwVSIFN.LS09qbuPjVxwIvOqC3i79lJ6hzHw722cwLRa4m', 'admin', 1, NULL, NULL, NOW() - INTERVAL '100 days', NOW()),
    (2, 'mod@wargame.local', 'moderator', '$2a$10$bsyMO/LWwVSIFN.LS09qbuPjVxwIvOqC3i79lJ6hzHw722cwLRa4m', 'user', 2, NULL, NULL, NOW() - INTERVAL '95 days', NOW()),
    (3, 'blocked@wargame.local', 'blocked-user', '$2a$10$bsyMO/LWwVSIFN.LS09qbuPjVxwIvOqC3i79lJ6hzHw722cwLRa4m', 'blocked', NULL, 'spam submissions', NOW() - INTERVAL '10 days', NOW() - INTERVAL '90 days', NOW());

INSERT INTO users (id, email, username, password_hash, role, affiliation_id, created_at, updated_at)
SELECT
    gs,
    format('user%s@wargame.local', gs - 3),
    format('user%s', gs - 3),
    '$2a$10$bsyMO/LWwVSIFN.LS09qbuPjVxwIvOqC3i79lJ6hzHw722cwLRa4m',
    'user',
    CASE
        WHEN gs % 5 = 0 THEN NULL
        ELSE ((gs % 4) + 1)
    END,
    NOW() - ((120 - gs) || ' days')::interval,
    NOW() - ((gs % 15) || ' hours')::interval
FROM generate_series(4, 34) AS gs;

INSERT INTO challenges (
    id,
    title,
    description,
    points,
    category,
    flag_hash,
    created_by_user_id,
    previous_challenge_id,
    file_key,
    file_name,
    file_uploaded_at,
    stack_enabled,
    stack_target_ports,
    stack_pod_spec,
    is_active,
    created_at
)
WITH generated AS (
    SELECT
        cid,
        CASE
            WHEN cid <= 14 THEN ((cid - 1) % 3) + 1
            WHEN cid <= 30 THEN ((cid - 15) % 3) + 4
            WHEN cid <= 50 THEN ((cid - 31) % 2) + 7
            WHEN cid <= 65 THEN 9
            ELSE 10
        END AS level
    FROM generate_series(1, 80) AS cid
)
SELECT
    g.cid,
    format('Challenge %03s', g.cid),
    format('This is dummy challenge #%s. Category rotation, prerequisite chain, and varied points are included for pagination/search/load testing.', g.cid),
    (g.level * 100) + ((g.cid % 3) * 20),
    (ARRAY['Web', 'Pwnable', 'Reversing', 'Crypto', 'Forensics', 'Cloud', 'Misc', 'Programming'])[((g.cid - 1) % 8) + 1],
    '$2a$10$bsyMO/LWwVSIFN.LS09qbuPjVxwIvOqC3i79lJ6hzHw722cwLRa4m',
    ((g.cid % 20) + 4),
    CASE
        WHEN g.cid % 9 = 0 THEN g.cid - 1
        WHEN g.cid % 13 = 0 THEN g.cid - 2
        ELSE NULL
    END,
    NULL,
    NULL,
    NULL,
    (g.cid % 7 = 0),
    CASE
        WHEN g.cid % 7 = 0 THEN '[{"container_port":8080,"protocol":"TCP"}]'::jsonb
        ELSE NULL
    END,
    CASE
        WHEN g.cid % 7 = 0 THEN 'apiVersion: v1\nkind: Pod\nmetadata:\n  name: challenge-' || g.cid || '\nspec:\n  containers:\n    - name: app\n      image: nginx:stable\n      ports:\n        - containerPort: 8080\n'
        ELSE NULL
    END,
    (g.cid % 11 <> 0),
    NOW() - ((90 - (g.cid % 60)) || ' days')::interval
FROM generated g;

-- Correct submissions (with deterministic first-blood per challenge)
WITH solved AS (
    SELECT
        u.id AS user_id,
        c.id AS challenge_id,
        (TIMESTAMP '2026-02-01 09:00:00' + (((c.id * 3 + u.id * 5) % 500) || ' minutes')::interval) AS submitted_at,
        ROW_NUMBER() OVER (
            PARTITION BY c.id
            ORDER BY ((c.id * 3 + u.id * 5) % 500), u.id
        ) AS rn
    FROM users u
    JOIN challenges c ON c.is_active = TRUE
    WHERE u.role = 'user'
      AND ((u.id * 17 + c.id * 11) % 19 = 0)
)
INSERT INTO submissions (user_id, challenge_id, provided, correct, is_first_blood, submitted_at)
SELECT
    s.user_id,
    s.challenge_id,
    format('FLAG{DUMMY_%s_%s}', s.user_id, s.challenge_id),
    TRUE,
    (s.rn = 1),
    s.submitted_at
FROM solved s;

-- Incorrect submissions for noise/load testing
INSERT INTO submissions (user_id, challenge_id, provided, correct, is_first_blood, submitted_at)
SELECT
    u.id,
    c.id,
    format('wrong-%s-%s', u.id, c.id),
    FALSE,
    FALSE,
    (TIMESTAMP '2026-02-01 08:00:00' + (((u.id * 7 + c.id * 13) % 720) || ' minutes')::interval)
FROM users u
JOIN challenges c ON c.is_active = TRUE
WHERE u.role = 'user'
  AND ((u.id * 7 + c.id * 13) % 31 = 0)
LIMIT 220;

-- Seed level votes from solved users so representative level is derived from votes.
-- Vote levels are deterministic 1..10 and comply with the "must solve before voting" rule.
INSERT INTO challenge_votes (user_id, challenge_id, level, created_at, updated_at)
SELECT
    s.user_id,
    s.challenge_id,
    (((s.user_id * 7 + s.challenge_id * 11) % 10) + 1) AS level,
    s.submitted_at + INTERVAL '5 minutes',
    s.submitted_at + INTERVAL '5 minutes'
FROM submissions AS s
JOIN users AS u ON u.id = s.user_id
WHERE s.correct = TRUE
  AND u.role = 'user'
  AND ((s.user_id * 13 + s.challenge_id * 17) % 4 <> 0);

-- Active stacks on stack-enabled challenges
INSERT INTO stacks (user_id, challenge_id, stack_id, status, node_public_ip, ports, ttl_expires_at, created_at, updated_at)
SELECT
    u.id,
    c.id,
    format('stack-%s-%s', u.id, c.id),
    CASE WHEN (u.id + c.id) % 3 = 0 THEN 'running' ELSE 'pending' END,
    format('10.10.%s.%s', (u.id % 200) + 1, (c.id % 200) + 1),
    jsonb_build_array(
        jsonb_build_object(
            'container_port', 8080,
            'protocol', 'TCP',
            'node_port', 30000 + ((u.id * 11 + c.id * 17) % 2000)
        )
    ),
    NOW() + (((u.id + c.id) % 6 + 1) || ' hours')::interval,
    NOW() - (((u.id + c.id) % 24) || ' hours')::interval,
    NOW() - (((u.id + c.id) % 6) || ' minutes')::interval
FROM users u
JOIN challenges c ON c.stack_enabled = TRUE AND c.is_active = TRUE
WHERE u.role = 'user'
  AND ((u.id * 5 + c.id * 3) % 9 = 0)
LIMIT 30;

-- Keep sequence values aligned after explicit IDs.
SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users), TRUE);
SELECT setval('affiliations_id_seq', (SELECT COALESCE(MAX(id), 1) FROM affiliations), TRUE);
SELECT setval('challenges_id_seq', (SELECT COALESCE(MAX(id), 1) FROM challenges), TRUE);
SELECT setval('submissions_id_seq', (SELECT COALESCE(MAX(id), 1) FROM submissions), TRUE);
SELECT setval('stacks_id_seq', (SELECT COALESCE(MAX(id), 1) FROM stacks), TRUE);

COMMIT;
