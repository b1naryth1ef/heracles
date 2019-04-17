def test_logout(user_session):
    r = user_session.post('/logout')
    assert r.status_code == 204

    r = user_session.get('/api/identity')
    assert r.status_code == 401


def test_validate(user_session, user_realm):
    r = user_session.get('/api/validate', headers={
        'X-Heracles-Realm': user_realm['name'],
    })
    assert r.status_code == 204
