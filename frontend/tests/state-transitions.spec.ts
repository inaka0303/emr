import { expect, test, type Page } from '@playwright/test';
import type { Fixture } from './fixtures/state-transitions';
import type { SOAPNote } from '../src/types/api';
import type { SoapDraftEntry } from '../src/hooks/useSoapDraftCache';

const baseProps = { patientName: 'テスト患者', encounterId: 1, interviewText: '問診' };
const storageKey = (id: number) => `emr:soap-editor-draft:general:current:${id}`;
const note: SOAPNote = {
  id: 11, encounter_id: 1, author: '', subjective: '保存済みS', objective: '', assessment: '', plan: '',
  is_slm_suggested: false, created_at: '2026-10-19 10:00:00', updated_at: '2026-10-19 10:00:00',
};

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>((done) => { resolve = done; });
  return { promise, resolve };
}

async function render(page: Page, fixture: Fixture) {
  await page.evaluate((value) => window.renderFixture(value), fixture);
}

async function settle(page: Page) {
  await page.evaluate(() => new Promise<void>((resolve) => requestAnimationFrame(() => resolve())));
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/slm/autocomplete', (route) => route.fulfill({ json: { data: { completion: '' } } }));
  await page.goto('/tests/fixtures/state-transitions.html');
  await page.waitForFunction(() => typeof window.renderFixture === 'function');
});

test('SOAP edits survive refetch and encounter switches; saving clears the recovery draft', async ({ page }) => {
  const props = { ...baseProps, aiEnabled: false, hasExistingSOAP: true, existingSOAPNote: note };
  await render(page, { kind: 'soap', props });
  await expect(page.getByRole('textbox').first()).toHaveValue('保存済みS');
  expect(await page.evaluate((key) => localStorage.getItem(key), storageKey(1))).toBeNull();

  await page.getByRole('textbox').first().fill('未保存の編集S');
  await render(page, { kind: 'soap', props: { ...props, existingSOAPNote: { ...note } } });
  await expect(page.getByRole('textbox').first()).toHaveValue('未保存の編集S');

  await render(page, { kind: 'soap', props: {
    ...props, encounterId: 2, existingSOAPNote: { ...note, id: 22, encounter_id: 2, subjective: '別受診S' },
  } });
  await expect(page.getByRole('textbox').first()).toHaveValue('別受診S');
  await render(page, { kind: 'soap', props });
  await expect(page.getByRole('textbox').first()).toHaveValue('未保存の編集S');

  let saved: unknown;
  await page.route('**/api/soap/11', (route) => {
    saved = route.request().postDataJSON();
    return route.fulfill({ json: { data: note } });
  });
  await page.getByRole('button', { name: '変更を保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '保存済み', exact: true })).toBeDisabled();
  expect(saved).toMatchObject({ subjective: '未保存の編集S' });
  expect(await page.evaluate((key) => localStorage.getItem(key), storageKey(1))).toBeNull();

  await render(page, { kind: 'soap', props: {
    ...props, existingSOAPNote: { ...note, subjective: '未保存の編集S', updated_at: '2026-10-19 10:01:00' },
  } });
  await expect(page.getByRole('button', { name: '保存済み', exact: true })).toBeDisabled();
  await expect(page.getByRole('textbox').first()).toHaveValue('未保存の編集S');
});

test('streaming SOAP preserves accepted and manual sections through updates and save', async ({ page }) => {
  const entry: SoapDraftEntry = {
    encounterId: 1, isLoading: true, suggestion: { subjective: '生成S', objective: '', assessment: '', plan: '' },
    meta: null, error: null, done: false, generation: 1,
  };
  await render(page, { kind: 'soap', props: { ...baseProps, draftEntry: entry } });
  await page.getByRole('button', { name: '採用', exact: true }).click();
  await page.getByRole('textbox').first().fill('医師編集S');

  const next = { ...entry, suggestion: { subjective: '更新S', objective: '生成O', assessment: '', plan: '' } };
  await render(page, { kind: 'soap', props: { ...baseProps, draftEntry: next } });
  await expect(page.getByRole('textbox').first()).toHaveValue('医師編集S');
  await page.getByRole('button', { name: '却下', exact: true }).click();
  await page.getByRole('textbox').nth(1).fill('手動O');

  await render(page, { kind: 'soap', props: { ...baseProps, draftEntry: {
    ...entry, done: true, isLoading: false,
    suggestion: { subjective: '再更新S', objective: '再更新O', assessment: '生成A', plan: '生成P' },
  } } });
  await page.getByRole('button', { name: '全ドラフトを採用' }).click();
  await expect(page.getByRole('textbox').nth(0)).toHaveValue('医師編集S');
  await expect(page.getByRole('textbox').nth(1)).toHaveValue('手動O');
  await expect(page.getByRole('textbox').nth(2)).toHaveValue('生成A');
  await expect(page.getByRole('textbox').nth(3)).toHaveValue('生成P');

  let saved: unknown;
  await page.route('**/api/encounters/1/soap', (route) => {
    saved = route.request().postDataJSON();
    return route.fulfill({ json: { data: note } });
  });
  await page.getByRole('button', { name: '保存', exact: true }).click();
  await expect(page.getByRole('button', { name: '保存済み', exact: true })).toBeDisabled();
  expect(saved).toEqual({ subjective: '医師編集S', objective: '手動O', assessment: '生成A', plan: '生成P' });
});

test('SOAP recovery survives reload and is isolated by experiment reset version', async ({ page }) => {
  const props = { ...baseProps, aiEnabled: false, experimentAttemptId: 'A01', draftStorageVersion: 'v1' };
  await render(page, { kind: 'soap', props });
  await page.getByRole('textbox').first().fill('復元するS');
  await page.reload();
  await page.waitForFunction(() => typeof window.renderFixture === 'function');
  await render(page, { kind: 'soap', props });
  await expect(page.getByRole('textbox').first()).toHaveValue('復元するS');
  await render(page, { kind: 'soap', props: { ...props, draftStorageVersion: 'v2' } });
  await expect(page.getByRole('textbox').first()).toHaveValue('');
});

test('SOAP generation errors leave manual input available', async ({ page }) => {
  await render(page, { kind: 'soap', props: { ...baseProps, draftEntry: {
    encounterId: 1, isLoading: false, suggestion: null, meta: null, error: '接続失敗', done: false, generation: 1,
  } } });
  await expect(page.getByText(/ドラフト生成エラー: 接続失敗/)).toBeVisible();
  await expect(page.getByRole('textbox')).toHaveCount(4);
  await page.getByRole('textbox').first().fill('手動記載');
  await expect(page.getByRole('button', { name: '保存', exact: true })).toBeEnabled();
});

test('interview changes clear stale data, ignore late responses, and clear loading for no encounter', async ({ page }) => {
  const oldResponse = deferred();
  const requested = deferred();
  await page.route('**/api/encounters/*/interviews', async (route) => {
    const old = route.request().url().includes('/1/');
    if (old) { requested.resolve(); await oldResponse.promise; }
    await route.fulfill({ json: { data: [{ id: old ? 1 : 2, raw_text: old ? '古い問診' : '現在の問診' }] } });
  });
  await render(page, { kind: 'interview', encounterId: 1 });
  await requested.promise;
  await expect(page.getByTestId('interview')).toContainText('"isLoading":true');
  await render(page, { kind: 'interview', encounterId: 2 });
  await expect(page.getByTestId('interview')).toContainText('現在の問診');
  const response = page.waitForResponse('**/api/encounters/1/interviews');
  oldResponse.resolve();
  await (await response).finished();
  await settle(page);
  await expect(page.getByTestId('interview')).not.toContainText('古い問診');

  await render(page, { kind: 'interview', encounterId: null });
  await expect(page.getByTestId('interview')).toHaveText('{"notes":[],"combinedText":"","isLoading":false,"error":null}');
  await page.route('**/api/encounters/3/interviews', (route) => route.fulfill({ status: 500, body: '取得失敗' }));
  await render(page, { kind: 'interview', encounterId: 3 });
  await expect(page.getByTestId('interview')).toContainText('取得失敗');
  await render(page, { kind: 'interview', encounterId: null });
  await expect(page.getByTestId('interview')).toHaveText('{"notes":[],"combinedText":"","isLoading":false,"error":null}');
});

test('admission summaries ignore stale generation after encounter or interview changes', async ({ page }) => {
  const oldGeneration = deferred();
  const requested = deferred();
  const generations: number[] = [];
  await page.route('**/api/encounters/*/admission-summary', (route) => route.fulfill({ json: { data: null } }));
  await page.route('**/api/slm/suggest/admission', async (route) => {
    const body = route.request().postDataJSON();
    generations.push(body.encounter_id);
    if (body.encounter_id === 1) { requested.resolve(); await oldGeneration.promise; }
    await route.fulfill({ json: { data: { text: `サマリ:${body.interview_text}` }, meta: { is_mock: true } } });
  });
  await render(page, { kind: 'admission', props: { encounterId: 1, interviewText: '古い入力' } });
  await requested.promise;
  await render(page, { kind: 'admission', props: { encounterId: 2, interviewText: '現在の入力' } });
  await expect(page.getByText('サマリ:現在の入力', { exact: true })).toBeVisible();
  const response = page.waitForResponse((res) => res.url().endsWith('/slm/suggest/admission') && res.request().postDataJSON().encounter_id === 1);
  oldGeneration.resolve();
  await (await response).finished();
  await settle(page);
  await expect(page.getByText('サマリ:現在の入力', { exact: true })).toBeVisible();
  expect(generations).toEqual([1, 2]);

  await page.getByRole('button', { name: '採用', exact: true }).click();
  await page.getByRole('textbox').fill('編集したサマリ');
  await render(page, { kind: 'admission', props: { encounterId: 2, interviewText: '修正後の入力' } });
  await expect(page.getByText('サマリ:修正後の入力', { exact: true })).toBeVisible();
  await expect(page.getByRole('textbox')).toHaveCount(0);
});

test('saved admission summaries can be edited and saved without generating again', async ({ page }) => {
  let saved: unknown;
  await page.route('**/api/slm/suggest/admission', () => { throw new Error('Unexpected generation'); });
  await page.route('**/api/encounters/1/admission-summary', (route) => {
    if (route.request().method() === 'POST') saved = route.request().postDataJSON();
    return route.fulfill({ json: { data: { content: '保存済みサマリ', is_slm_draft: false } } });
  });
  await render(page, { kind: 'admission', props: { encounterId: 1, interviewText: '問診' } });
  await expect(page.getByRole('textbox')).toHaveValue('保存済みサマリ');
  await page.getByRole('textbox').fill('更新したサマリ');
  await page.getByRole('button', { name: '入院時サマリを保存' }).click();
  await expect(page.getByText('保存しました ✓')).toBeVisible();
  expect(saved).toEqual({ content: '更新したサマリ', author: '', is_slm_draft: false });
});
