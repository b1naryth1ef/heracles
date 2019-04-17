import pytest


def test_get_tokens(user_session):
    r = user_session.get('/api/tokens')
    assert r.status_code == 200
    assert r.json()['tokens'] == [user_session.token]


@pytest.mark.parametrize('admin', [True, False])
def test_create_token(session, user_session, admin_session, random_string, admin):
    name = random_string(32)

    if admin:
        r = admin_session.post('/api/tokens', data={
            'name': name,
            'user_id': user_session.user_id,
        })
    else:
        r = user_session.post('/api/tokens', data={
            'name': name,
        })

    assert r.status_code == 200

    data = r.json()
    assert data['name'] == name

    r = session.get('/api/identity', headers={
        'Authorization': data['token'],
    })
    assert r.status_code == 200
    assert r.json()['username'] == user_session.username

    r = user_session.get('/api/tokens')
    assert r.status_code == 200
    assert len(r.json()['tokens']) == 2


@pytest.mark.parametrize('admin', [True, False])
def test_patch_token(admin, user_token, user_session, admin_session, random_string):
    new_name = random_string(32)

    r = (admin_session if admin else user_session).patch(f"/api/tokens/{user_token['id']}", data={
        'name': new_name,
    })
    assert r.status_code == 200
    assert r.json()['name'] == new_name


@pytest.mark.parametrize('admin', [True, False])
def test_delete_token(admin, user_token, user_session, admin_session):
    r = user_session.get('/api/tokens')
    assert r.status_code == 200
    assert user_token in r.json()['tokens']

    r = (admin_session if admin else user_session).delete(f"/api/tokens/{user_token['id']}")
    assert r.status_code == 204

    r = user_session.get('/api/tokens')
    assert r.status_code == 200
    assert r.json()['tokens'] == [user_session.token]
