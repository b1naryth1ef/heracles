def test_get_identity(admin_session, user_session):
    r = admin_session.get('/api/identity')
    assert r.status_code == 200
    assert r.json()['id'] == 1
    assert r.json()['username'] == 'admin'

    r = user_session.get('/api/identity')
    assert r.status_code == 200


def test_patch_identity(session, user_session):
    r = user_session.patch('/api/identity', data={
        'password': 'test'
    })
    assert r.status_code == 204

    r = session.post('/login', data={
        'username': user_session.username,
        'password': 'test',
    })
    assert r.status_code == 204

    r = session.post('/login', data={
        'username': user_session.username,
        'password': user_session.password,
    })
    assert r.status_code == 400
    assert r.content == b'Bad password\n'
