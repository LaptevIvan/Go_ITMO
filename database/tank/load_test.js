import http from 'k6/http';
import {check, sleep} from 'k6';

export const options = {
    vus: 20,
    duration: '10s',
};

const urlAuthor = 'http://localhost:8080/v1/library/author';
const urlBook = 'http://localhost:8080/v1/library/book'
const authorIds = [
    'e6672056-49ee-4aba-a9f0-21813b2963a3',
    'de41b5cb-859a-4e82-8a15-5e83a609d510',
    'fbb90d1a-5616-4e0f-9242-4b1015c94cbf',
    'a37751e6-b8db-465e-bb8b-5182e5799fc3',
    '0f84bfb0-cda5-41f6-851d-25cf7b35e5d4',
];

export default function () {
    const payloadAuthor = JSON.stringify({
        name: `author${__VU}${__ITER}`
    });
    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    var res = http.post(urlAuthor, payloadAuthor, params);

    check(res, {
        'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    });

    const idAuthor = res.json().id

    const payloadBook = JSON.stringify({
        name: `book-${__VU}-${__ITER}`,
        authorIds: [
            idAuthor
        ]
    });

    res = http.post(urlBook, payloadBook, params);
    check(res, {
        'status is 200 or 201': (r) => r.status === 200 || r.status === 201,
    });

    sleep(1);
}

