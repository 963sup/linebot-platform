import {test} from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';

const source = await readFile(new URL('../internal/adapter/inbound/http/miniapp/assets/app.js', import.meta.url), 'utf8');
const settle = () => new Promise(resolve => setImmediate(resolve));

test('consent recovery requires verified server session before enabling location', async () => {
  const elements = Object.fromEntries(['status', 'locate', 'location', 'login'].map(id => [id, {
    disabled: true, hidden: true, addEventListener(event, callback) { this[event] = callback; }
  }]));
  globalThis.document = {querySelector: selector => elements[selector.slice(1)]};
  let granted = false, requests = 0, accepted = false;
  globalThis.liff = {
    init: async () => {}, isLoggedIn: () => true,
    permission: {query: async () => ({state: granted ? 'granted' : 'prompt'}),
      requestAll: async () => { granted = true; }},
    getIDToken: () => granted ? 'fake-token' : null
  };
  globalThis.fetch = async path => {
    if (path.endsWith('config.json')) return {ok: true, json: async () => ({liffId: 'test'})};
    requests++;
    return {ok: accepted, status: 401, json: async () => ({authenticated: true})};
  };
  await import('data:text/javascript,' + encodeURIComponent(source));
  await settle();
  assert.equal(requests, 0);
  assert.equal(elements.locate.disabled, true);
  assert.equal(elements.login.hidden, false);
  await elements.login.click();
  assert.equal(granted, true);
  assert.equal(requests, 1);
  assert.equal(elements.locate.disabled, true);
  accepted = true;
  await elements.login.click();
  assert.equal(elements.locate.disabled, false);
  assert.equal(elements.login.hidden, true);
});
