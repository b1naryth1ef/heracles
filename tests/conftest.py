import os
import time
import random
import string
import subprocess

import pytest
import requests_unixsocket


def get_random_string(size):
    return ''.join([random.choice(string.ascii_letters) for _ in range(size)])


class SessionWithUrlBase(requests_unixsocket.Session):
    def __init__(self, *args, url_base=None, **kwargs):
        super(SessionWithUrlBase, self).__init__(*args, **kwargs)
        self.url_base = url_base

    def request(self, method, url, **kwargs):
        modified_url = self.url_base + url
        return super(SessionWithUrlBase, self).request(method, modified_url, **kwargs)


@pytest.fixture(scope='session')
def random_string():
    return get_random_string


@pytest.fixture(scope='session')
def heracles(request):
    proc = subprocess.Popen(['./heracles'], env={
        'WEB_BIND': 'unix://testing.sock',
        'DB_PATH': ':memory:',
        'SECURITY_SECRET': get_random_string(64),
        'SECURITY_BCRYPT_DIFFICULTY': '1',
    })

    time.sleep(1)

    def finalize():
        proc.kill()
        os.remove('testing.sock')

    request.addfinalizer(finalize)
    return 'http+unix://testing.sock'


@pytest.fixture(scope='function')
def session(heracles):
    session = SessionWithUrlBase(url_base=heracles)
    return session


@pytest.fixture(scope='session')
def admin_session(heracles):
    session = SessionWithUrlBase(url_base=heracles)
    resp = session.post('/login', data={
        'username': 'admin',
        'password': 'admin',
    })
    assert resp.status_code == 204
    return session


@pytest.fixture(scope='function')
def user_session(admin_session, heracles):
    return create_user_session(admin_session, heracles, False)


@pytest.fixture(scope='function')
def user_session_with_password(admin_session, heracles):
    return create_user_session(admin_session, heracles, True)


def create_user_session(admin_session, heracles, with_password):
    session = SessionWithUrlBase(url_base=heracles)

    username = get_random_string(32)
    password = get_random_string(32) if with_password else ''

    resp = admin_session.post('/api/users', data={
        'username': username,
        'password': password,
    })
    assert resp.status_code == 200
    user_id = resp.json()['id']

    resp = admin_session.post('/api/tokens', data={
        'name': get_random_string(32),
        'user_id': user_id,
    })
    assert resp.status_code == 200

    session.user_id = user_id
    session.username = username
    session.password = password
    session.token = resp.json()

    session.headers['Authorization'] = session.token['token']

    return session


@pytest.fixture(scope='function')
def user_realm(user_session, admin_session):
    realm_name = get_random_string(32)

    resp = admin_session.post('/api/realms', data={
        'name': realm_name,
    })
    assert resp.status_code == 200
    realm = resp.json()

    resp = admin_session.post(f"/api/realms/{realm['id']}/grants", data={
        'user_id': user_session.user_id,
    })
    assert resp.status_code == 200
    return realm


@pytest.fixture(scope='function')
def user_token(user_session):
    r = user_session.post('/api/tokens', data={
        'name': get_random_string(32),
    })
    assert r.status_code == 200
    return r.json()
