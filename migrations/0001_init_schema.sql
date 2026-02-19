CREATE TABLE IF NOT EXISTS users (
    user_id BIGINT PRIMARY KEY,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    nickname TEXT,
    registration_date TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS courses (
    id SERIAL PRIMARY KEY,
    course_id TEXT UNIQUE NOT NULL,
    course_name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id BIGINT PRIMARY KEY,
    role TEXT DEFAULT 'student',
    curator_id BIGINT REFERENCES users(user_id),
    course_id TEXT REFERENCES courses(course_id),
    admin_notifications BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (user_id) REFERENCES users (user_id)
    );

CREATE TABLE IF NOT EXISTS curator_courses (
    id SERIAL PRIMARY KEY,
    curator_id BIGINT REFERENCES users(user_id),
    course_id TEXT REFERENCES courses(course_id),
    UNIQUE(curator_id, course_id)
);

CREATE TABLE IF NOT EXISTS subtask_mappings (
    id SERIAL PRIMARY KEY,
    main_task TEXT,
    subtask_code TEXT,
    subtask_name TEXT,
    subtask_type TEXT,
    display_order INTEGER,
    UNIQUE(main_task, subtask_code)
);

CREATE TABLE IF NOT EXISTS submissions (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(user_id),
    curator_id BIGINT REFERENCES users(user_id),
    submission_type TEXT,
    task_number TEXT,
    file_paths TEXT[],
    original_names TEXT[],
    comment TEXT,
    submission_date TIMESTAMP DEFAULT NOW(),
    status TEXT DEFAULT 'pending'
);

CREATE TABLE IF NOT EXISTS daily_status (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(user_id),
    date DATE,
    homework_status TEXT DEFAULT 'not_done',
    notes_status TEXT DEFAULT 'not_done',
    homework_reason TEXT,
    notes_reason TEXT,
    homework_reason_date TIMESTAMP,
    notes_reason_date TIMESTAMP,
    UNIQUE(user_id, date)
);

CREATE TABLE IF NOT EXISTS user_tasks_status (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(user_id),
    task_number TEXT,
    homework_status TEXT DEFAULT 'not_done',
    notes_status TEXT DEFAULT 'not_done',
    homework_reason TEXT,
    notes_reason TEXT,
    homework_reason_date TIMESTAMP,
    notes_reason_date TIMESTAMP,
    last_updated TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, task_number)
);

CREATE TABLE IF NOT EXISTS user_paths (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(user_id),
    file_path TEXT,
    task_number TEXT,
    submission_type TEXT,
    UNIQUE(user_id, file_path)
);

CREATE TABLE IF NOT EXISTS weekly_reports (
    id SERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(user_id),
    week_start_date DATE,
    report_text TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, week_start_date)
);

CREATE TABLE IF NOT EXISTS developer_attachments (
    id SERIAL PRIMARY KEY,
    developer_id BIGINT REFERENCES users(user_id),
    curator_id BIGINT REFERENCES users(user_id),
    course_id TEXT REFERENCES courses(course_id),
    attachment_date TIMESTAMP DEFAULT NOW(),
    UNIQUE(developer_id, curator_id, course_id)
);

CREATE TABLE IF NOT EXISTS cms_contents (
    key VARCHAR(255) PRIMARY KEY,
    content TEXT NOT NULL,
    description TEXT
);



INSERT INTO courses (course_id, course_name) VALUES
    ('year', 'Годовой'),
    ('half_year', 'Полугодовой'),
    ('three_month', '3-х месячный курс')
ON CONFLICT (course_id) DO UPDATE
    SET course_name = EXCLUDED.course_name;

INSERT INTO subtask_mappings (main_task, subtask_code, subtask_name, subtask_type, display_order)
SELECT gs::text, gs::text, gs::text || ' задание ЕГЭ', 'normal', gs * 10
FROM generate_series(1, 27) AS gs
    ON CONFLICT DO NOTHING;

INSERT INTO subtask_mappings (main_task, subtask_code, subtask_name, subtask_type, display_order) VALUES
    ('1', '1.1', '1 ЕГЭ – часть 1', 'normal', 11),
    ('1', '1.2', '1 ЕГЭ – часть 2', 'normal', 12),
    ('24', '24.1', '24 ЕГЭ – классический (ч.1)', 'normal', 241),
    ('26', '26.1', '26 ЕГЭ – Python (ч.1)', 'normal', 261),
    ('27', '27.1', '27 задание ЕГЭ – часть 1', 'normal', 271),
    ('Практика', 'PR.0', 'Практика', 'practice', 1000),
    ('Практика', 'PR.2', 'Практика: Excel', 'practice', 1002),
    ('Python', 'PY.1', 'Python (введение)', 'practice', 2001)
ON CONFLICT DO NOTHING;