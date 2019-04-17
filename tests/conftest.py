import os
import time
import pytest
import subprocess
import requests_unixsocket
import random
import string


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
        'BIND': 'unix://testing.sock',
        'DB_PATH': ':memory:',
        'DIFFICULTY': '1',
    })

    def finalize():
        proc.kill()
        os.remove('testing.sock')

    # TODO: Be more deterministic about startup
    time.sleep(1)

    request.addfinalizer(finalize)
    return 'http+unix://testing.sock'


@pytest.fixture(scope='function')
def session(heracles):
    session = SessionWithUrlBase(url_base=heracles)
    return session


@pytest.fixture(scope='function')
def admin_session(heracles):
    session = SessionWithUrlBase(url_base=heracles)
    resp = session.post('/login', data={
        'username': 'admin',
        'password': 'admin',
    })
    assert resp.status_code == 204
    return session


@pytest.fixture(scope='function')
def user_session(admin_session, session):
    username = get_random_string(32)
    password = get_random_string(32)

    resp = admin_session.post('/api/users', data={
        'username': username,
        'password': password,
    })
    assert resp.status_code == 200
    user_id = resp.json()['id']

    resp = session.post('/login', data={
        'username': username,
        'password': password,
    })

    session.user_id = user_id
    session.username = username
    session.password = password

    assert resp.status_code == 204
    return session


@pytest.fixture(scope='function')
def user_realm(user_session, admin_session):
    realm_name = get_random_string(32)

    resp = admin_session.post('/api/realms', data={
        'name': realm_name,
    })
    assert resp.status_code == 200
    realm = resp.json()

    resp = admin_session.post('/api/realms/grants', data={
        'realm_id': realm['id'],
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
