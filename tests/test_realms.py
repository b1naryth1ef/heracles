import pytest


def test_get_realms(admin_session):
    r = admin_session.get('/api/realms')
    assert r.status_code == 200
    assert isinstance(r.json()['realms'], list)


def test_create_realm(admin_session, random_string):
    name = random_string(32)

    r = admin_session.post('/api/realms', data={
        'name': name,
    })
    assert r.status_code == 200
    realm = r.json()
    assert realm['name'] == name

    r = admin_session.get('/api/realms')
    assert realm in r.json()['realms']


@pytest.mark.parametrize('alias', [None, 'test_alias'])
def test_create_realm_grant(alias, admin_session, user_session, random_string):
    r = admin_session.post('/api/realms', data={
        'name': random_string(32),
    })
    assert r.status_code == 200
    realm = r.json()

    r = user_session.get('/api/validate', headers={
        'X-Heracles-Realm': realm['name'],
    })
    assert r.status_code == 401

    r = admin_session.post(f"/api/realms/{realm['id']}/grants", data={
        'user_id': user_session.user_id,
        'alias': alias,
    })
    assert r.status_code == 200

    r = user_session.get('/api/validate', headers={
        'X-Heracles-Realm': realm['name'],
    })
    assert r.status_code == 204
