def test_login(user_session_with_password, session):
    r = session.post('/login', data={
        'username': user_session_with_password.username,
        'password': user_session_with_password.password,
    })
    assert r.status_code == 204

    r = session.get('/api/identity')
    assert r.status_code == 200


def test_logout(user_session_with_password, session):
    r = session.post('/login', data={
        'username': user_session_with_password.username,
        'password': user_session_with_password.password,
    })
    assert r.status_code == 204

    r = session.post('/logout')
    assert r.status_code == 204

    r = session.get('/api/identity')
    assert r.status_code == 401


def test_validate(user_session, user_realm):
    r = user_session.get('/api/validate', headers={
        'X-Heracles-Realm': user_realm['name'],
    })
    assert r.status_code == 204
