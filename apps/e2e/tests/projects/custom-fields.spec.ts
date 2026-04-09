// spec: features/projects/custom-fields.feature
// seed: tests/seed.spec.ts

import { test, expect, type Page, type APIRequestContext } from '@playwright/test';

const BASE_URL = process.env.E2E_BASE_URL ?? 'http://localhost';
const USERNAME = process.env.E2E_USERNAME ?? 'admin';
const PASSWORD = process.env.E2E_PASSWORD ?? 'e2e-admin-password';
const TEST_PROJECT_PREFIX = 'E2E_CFIELD_';
const RUN_ID = Date.now().toString(36).slice(-5).toUpperCase();

// ── API Helpers ───────────────────────────────────────────────────────────────

async function authRequest(request: APIRequestContext): Promise<void> {
  await request.post(`${BASE_URL}/api/v1/auth/login`, {
    data: { username: USERNAME, password: PASSWORD, rememberMe: false },
  });
}

async function cleanupTestProjects(request: APIRequestContext): Promise<void> {
  await authRequest(request);

  const allProjects: Array<{ id: string; name: string }> = [];
  let page = 1;

  while (true) {
    const listResp = await request.get(`${BASE_URL}/api/v1/projects?page=${page}&page_size=100`);
    if (!listResp.ok()) break;
    const body = await listResp.json();
    const items: Array<{ id: string; name: string }> = body?.data?.items ?? [];
    if (items.length === 0) break;
    allProjects.push(...items);
    const { page: currentPage, page_size, total } = body.data as { page: number; page_size: number; total: number };
    if (currentPage * page_size >= total) break;
    page++;
  }

  await Promise.all(
    allProjects
      .filter((p) => p.name.startsWith(TEST_PROJECT_PREFIX))
      .map((p) => request.delete(`${BASE_URL}/api/v1/projects/${p.id}`)),
  );
}

async function createProject(request: APIRequestContext, name: string): Promise<string> {
  const resp = await request.post(`${BASE_URL}/api/v1/projects`, {
    data: { name },
  });
  const body = await resp.json();
  return body.data.id as string;
}

async function createCustomField(
  request: APIRequestContext,
  projectId: string,
  opts: {
    display_name: string;
    field_key?: string;
    field_type?: string;
    options?: string[];
    is_required?: boolean;
  },
): Promise<string> {
  const fieldKey = opts.field_key ?? opts.display_name.toLowerCase().replace(/\s+/g, '_').replace(/[^a-z0-9_]/g, '');
  const resp = await request.post(`${BASE_URL}/api/v1/projects/${projectId}/custom-fields`, {
    data: {
      display_name: opts.display_name,
      field_key: fieldKey,
      field_type: opts.field_type ?? 'text',
      options: opts.options ?? [],
      is_required: opts.is_required ?? false,
    },
  });
  const body = await resp.json();
  return body.data.id as string;
}

// ── Browser Helpers ───────────────────────────────────────────────────────────

const signIn = async (page: Page) => {
  await page.goto(`${BASE_URL}/`);
  await page.getByRole('textbox', { name: 'Username' }).fill(USERNAME);
  await page.getByRole('textbox', { name: 'Password' }).fill(PASSWORD);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page.getByRole('heading', { name: /Good (morning|afternoon|evening)/i })).toBeVisible();
};

const navigateToProjectSettings = async (page: Page, projectId: string) => {
  await page.goto(`${BASE_URL}/projects/${projectId}/settings`);
  await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible();
};

const openCustomFieldsSection = async (page: Page) => {
  await page.getByRole('button', { name: 'Custom Fields' }).click();
  await expect(page.getByRole('heading', { name: 'Custom Fields', level: 3 })).toBeVisible();
};

// ── Tests ─────────────────────────────────────────────────────────────────────

test.describe('Custom field management', () => {
  // ---------------------------------------------------------------------------
  // Rule: Viewing custom fields
  // ---------------------------------------------------------------------------

  test.describe('Viewing custom fields', () => {
    let projectId: string;

    test.beforeEach(async ({ request, context }) => {
      await cleanupTestProjects(request);
      projectId = await createProject(request, `E2E_CFIELD_VIEW_${RUN_ID}`);
      await context.clearCookies();
      await context.clearPermissions();
    });

    test.afterEach(async ({ request }) => {
      await cleanupTestProjects(request);
    });

    test('Custom Fields section is reachable from the settings sidebar', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);

      await page.getByRole('button', { name: 'Custom Fields' }).click();

      await expect(page.getByRole('heading', { name: 'Custom Fields', level: 3 })).toBeVisible();
      await expect(page.getByText(/extend tasks|additional data/i)).toBeVisible();
    });

    test('New project shows an empty state with no custom fields', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await expect(page.getByText(/No custom fields yet/i)).toBeVisible();
      await expect(page.getByText(/Create first field|create.*field/i)).toBeVisible();
    });

    test('Custom fields table shows the expected columns once fields exist', async ({ page, request }) => {
      await createCustomField(request, projectId, { display_name: 'E2E Priority Score' });

      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await expect(page.getByRole('columnheader', { name: 'Display Name' })).toBeVisible();
      await expect(page.getByRole('columnheader', { name: 'Field Key' })).toBeVisible();
      await expect(page.getByRole('columnheader', { name: 'Type' })).toBeVisible();
      await expect(page.getByRole('columnheader', { name: 'Required' })).toBeVisible();
    });

    test('Each custom field row has Edit and Delete action buttons', async ({ page, request }) => {
      await createCustomField(request, projectId, { display_name: 'E2E Priority Score' });

      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await expect(page.getByRole('button', { name: 'Edit field' }).first()).toBeVisible();
      await expect(page.getByRole('button', { name: 'Delete field' }).first()).toBeVisible();
    });

    test('"New custom field" button is visible on the Custom Fields section', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await expect(page.getByRole('button', { name: 'New custom field' })).toBeVisible();
    });

    test('Field key is displayed in monospace style', async ({ page, request }) => {
      await createCustomField(request, projectId, { display_name: 'E2E Priority Score' });

      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      // The field key cell should use font-mono class
      const row = page.getByRole('row').filter({ hasText: 'E2E Priority Score' });
      const keyCell = row.locator('td').nth(1);
      await expect(keyCell).toHaveClass(/font-mono/);
    });
  });

  // ---------------------------------------------------------------------------
  // Rule: Creating a custom field
  // ---------------------------------------------------------------------------

  test.describe('Creating a custom field', () => {
    let projectId: string;

    test.beforeEach(async ({ request, context }) => {
      await cleanupTestProjects(request);
      projectId = await createProject(request, `E2E_CFIELD_CREATE_${RUN_ID}`);
      await context.clearCookies();
      await context.clearPermissions();
    });

    test.afterEach(async ({ request }) => {
      await cleanupTestProjects(request);
    });

    test('Opening the create-custom-field dialog', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await expect(dialog).toBeVisible();
      await expect(dialog.getByRole('textbox', { name: /Display name/i })).toBeVisible();
      await expect(dialog.getByRole('textbox', { name: /Field key/i })).toBeVisible();
      await expect(dialog.getByRole('switch', { name: /Required/i })).toBeVisible();
    });

    test('"Create field" button is disabled while the display name is empty', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await expect(dialog.getByRole('button', { name: 'Create field' })).toBeDisabled();
    });

    test('"Create field" button becomes enabled after typing a display name', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Sprint Points');

      await expect(dialog.getByRole('button', { name: 'Create field' })).toBeEnabled();
    });

    test('Typing a display name auto-generates the field key', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('Sprint Points');

      await expect(dialog.getByRole('textbox', { name: /Field key/i })).toHaveValue('sprint_points');
    });

    test('Auto-generated field key can be manually overridden', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('Sprint Points');
      const keyInput = dialog.getByRole('textbox', { name: /Field key/i });
      await keyInput.clear();
      await keyInput.fill('sp_pts');

      await expect(keyInput).toHaveValue('sp_pts');
    });

    test('Field type selector lists all available types', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await expect(dialog.getByRole('button', { name: 'Text' })).toBeVisible();
      await expect(dialog.getByRole('button', { name: 'Number' })).toBeVisible();
      await expect(dialog.getByRole('button', { name: 'Date' })).toBeVisible();
      await expect(dialog.getByRole('button', { name: 'Checkbox' })).toBeVisible();
      await expect(dialog.getByRole('button', { name: 'Select' })).toBeVisible();
    });

    test('Default field type is Text', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      // The Text button should have the selected styling (primary class)
      await expect(dialog.getByRole('button', { name: 'Text' })).toHaveClass(/text-primary|bg-primary/);
    });

    test('Options editor is hidden for non-Select field types', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('button', { name: 'Number' }).click();

      await expect(dialog.getByLabel('Options')).not.toBeVisible();
    });

    test('Options editor appears when the Select field type is chosen', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('button', { name: 'Select' }).click();

      await expect(dialog.getByLabel('Options')).toBeVisible();
    });

    test('Adding an option to a Select field', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('button', { name: 'Select' }).click();
      await dialog.getByPlaceholder('Add option…').fill('High');
      await dialog.getByRole('button', { name: 'Add' }).click();

      await expect(dialog.locator('input[value="High"]')).toBeVisible();
    });

    test('Removing an option from a Select field', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('button', { name: 'Select' }).click();
      await dialog.getByPlaceholder('Add option…').fill('Unwanted');
      await dialog.getByRole('button', { name: 'Add' }).click();
      // Remove it via the X button next to the option input
      await dialog.locator('input[value="Unwanted"]').locator('..').getByRole('button').click();

      await expect(dialog.locator('input[value="Unwanted"]')).not.toBeVisible();
    });

    test('Creating a Text field succeeds', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Notes');
      await dialog.getByRole('button', { name: 'Text' }).click();
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Notes' });
      await expect(row).toBeVisible();
      await expect(row).toContainText('Text');
    });

    test('Creating a Number field succeeds', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Story Points');
      await dialog.getByRole('button', { name: 'Number' }).click();
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Story Points' });
      await expect(row).toBeVisible();
      await expect(row).toContainText('Number');
    });

    test('Creating a Date field succeeds', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Due Date');
      await dialog.getByRole('button', { name: 'Date' }).click();
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Due Date' });
      await expect(row).toBeVisible();
      await expect(row).toContainText('Date');
    });

    test('Creating a Checkbox field succeeds', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Blocked');
      await dialog.getByRole('button', { name: 'Checkbox' }).click();
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Blocked' });
      await expect(row).toBeVisible();
      await expect(row).toContainText('Checkbox');
    });

    test('Creating a Select field with options succeeds', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Severity');
      await dialog.getByRole('button', { name: 'Select' }).click();
      for (const opt of ['Low', 'Medium', 'High']) {
        await dialog.getByPlaceholder('Add option…').fill(opt);
        await dialog.getByRole('button', { name: 'Add' }).click();
      }
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Severity' });
      await expect(row).toBeVisible();
      await expect(row).toContainText('Select');
    });

    test('Creating a Required field shows "Yes" in the Required column', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Required Field');
      await dialog.getByRole('switch', { name: /Required/i }).click();
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Required Field' });
      await expect(row).toContainText('Yes');
    });

    test('Creating an optional field shows "No" in the Required column', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Optional Field');
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Optional Field' });
      await expect(row).toContainText('No');
    });

    test('Created field key matches the auto-generated slug', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Sprint Goal');
      await dialog.getByRole('button', { name: 'Create field' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Sprint Goal' });
      await expect(row).toContainText('e2e_sprint_goal');
    });

    test('Cancelling the create dialog discards changes', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Should Not Exist');
      await dialog.getByRole('button', { name: 'Cancel' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Should Not Exist' })).not.toBeVisible();
    });

    test('Closing the create dialog with the Close button discards changes', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Should Not Exist via X');
      // Close via the X button on the dialog
      await dialog.getByRole('button', { name: 'Close' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Should Not Exist via X' })).not.toBeVisible();
    });

    test('Duplicate field key within the same project is rejected', async ({ page, request }) => {
      await createCustomField(request, projectId, {
        display_name: 'Existing Field',
        field_key: 'e2e_dup_key',
      });

      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'New custom field' }).click();

      const dialog = page.getByRole('dialog', { name: 'Create custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).fill('E2E Duplicate');
      const keyInput = dialog.getByRole('textbox', { name: /Field key/i });
      await keyInput.clear();
      await keyInput.fill('e2e_dup_key');
      await dialog.getByRole('button', { name: 'Create field' }).click();

      // Dialog stays open and shows an error
      await expect(dialog).toBeVisible();
      await expect(dialog.getByText(/already in use|already exists/i)).toBeVisible();
    });
  });

  // ---------------------------------------------------------------------------
  // Rule: Editing a custom field
  // ---------------------------------------------------------------------------

  test.describe('Editing a custom field', () => {
    let projectId: string;

    test.beforeEach(async ({ request, context }) => {
      await cleanupTestProjects(request);
      projectId = await createProject(request, `E2E_CFIELD_EDIT_${RUN_ID}`);
      await createCustomField(request, projectId, {
        display_name: 'E2E Edit Me Field',
        field_key: 'e2e_edit_me_field',
        field_type: 'select',
        options: ['Alpha', 'Beta'],
      });
      await context.clearCookies();
      await context.clearPermissions();
    });

    test.afterEach(async ({ request }) => {
      await cleanupTestProjects(request);
    });

    test('Opening the edit dialog pre-fills existing values', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      await expect(dialog).toBeVisible();
      await expect(dialog.getByRole('textbox', { name: /Display name/i })).toHaveValue('E2E Edit Me Field');
      await expect(dialog.locator('input[value="e2e_edit_me_field"]')).toBeVisible();
    });

    test('Field type is not editable in the edit dialog', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      // The field type button should be disabled
      const fieldTypeBtn = dialog.locator('button[disabled]').filter({ hasText: /select/i });
      await expect(fieldTypeBtn).toBeVisible();
    });

    test('Field key is immutable in the edit dialog', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      const keyInput = dialog.locator('input[value="e2e_edit_me_field"][disabled]');
      await expect(keyInput).toBeVisible();
      await expect(keyInput).toBeDisabled();
    });

    test('Saving a new display name updates the field in the table', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      const nameInput = dialog.getByRole('textbox', { name: /Display name/i });
      await nameInput.clear();
      await nameInput.fill('E2E Renamed Field');
      await dialog.getByRole('button', { name: 'Save changes' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Renamed Field' })).toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Edit Me Field' })).not.toBeVisible();
    });

    test('Clearing the display name disables the save button', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      await dialog.getByRole('textbox', { name: /Display name/i }).clear();

      await expect(dialog.getByRole('button', { name: 'Save changes' })).toBeDisabled();
    });

    test('Options editor is visible for a Select field in edit mode', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      await expect(dialog.getByLabel('Options')).toBeVisible();
      await expect(dialog.locator('input[value="Alpha"]')).toBeVisible();
      await expect(dialog.locator('input[value="Beta"]')).toBeVisible();
    });

    test('Adding an option to an existing Select field saves correctly', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      await dialog.getByPlaceholder('Add option…').fill('Gamma');
      await dialog.getByRole('button', { name: 'Add' }).click();
      await dialog.getByRole('button', { name: 'Save changes' }).click();

      await expect(dialog).not.toBeVisible();
    });

    test('Toggling the Required flag updates the Required column', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      await dialog.getByRole('switch', { name: /Required/i }).click();
      await dialog.getByRole('button', { name: 'Save changes' }).click();

      await expect(dialog).not.toBeVisible();
      const row = page.getByRole('row').filter({ hasText: 'E2E Edit Me Field' });
      await expect(row).toContainText('Yes');
    });

    test('Cancelling the edit dialog discards changes', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Edit field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Edit custom field' });
      const nameInput = dialog.getByRole('textbox', { name: /Display name/i });
      await nameInput.clear();
      await nameInput.fill('E2E Should Not Save');
      await dialog.getByRole('button', { name: 'Cancel' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Edit Me Field' })).toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Should Not Save' })).not.toBeVisible();
    });
  });

  // ---------------------------------------------------------------------------
  // Rule: Deleting a custom field
  // ---------------------------------------------------------------------------

  test.describe('Deleting a custom field', () => {
    let projectId: string;

    test.beforeEach(async ({ request, context }) => {
      await cleanupTestProjects(request);
      projectId = await createProject(request, `E2E_CFIELD_DELETE_${RUN_ID}`);
      await createCustomField(request, projectId, { display_name: 'E2E Delete Me Field' });
      await context.clearCookies();
      await context.clearPermissions();
    });

    test.afterEach(async ({ request }) => {
      await cleanupTestProjects(request);
    });

    test('Opening the delete dialog shows a confirmation message', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Delete field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Delete custom field' });
      await expect(dialog).toBeVisible();
      await expect(dialog.getByText('E2E Delete Me Field')).toBeVisible();
      await expect(dialog.getByText(/cannot be undone/i)).toBeVisible();
      await expect(dialog.getByText(/data.*lost|lost|will be lost/i)).toBeVisible();
    });

    test('Confirming deletion removes the field from the table', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Delete field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Delete custom field' });
      await dialog.getByRole('button', { name: 'Delete field' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Delete Me Field' })).not.toBeVisible();
    });

    test('Confirming deletion of the last field returns to the empty state', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Delete field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Delete custom field' });
      await dialog.getByRole('button', { name: 'Delete field' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByText(/No custom fields yet/i)).toBeVisible();
    });

    test('Cancelling the delete dialog keeps the field in the table', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Delete field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Delete custom field' });
      await dialog.getByRole('button', { name: 'Cancel' }).click();

      await expect(dialog).not.toBeVisible();
      await expect(page.getByRole('cell', { name: 'E2E Delete Me Field' })).toBeVisible();
    });

    test('Closing the delete dialog with the Close button keeps the field', async ({ page }) => {
      await signIn(page);
      await navigateToProjectSettings(page, projectId);
      await openCustomFieldsSection(page);

      await page.getByRole('button', { name: 'Delete field' }).first().click();

      const dialog = page.getByRole('dialog', { name: 'Delete custom field' });
      await dialog.getByRole('button', { name: 'Close' }).click();

      await expect(page.getByRole('cell', { name: 'E2E Delete Me Field' })).toBeVisible();
    });
  });
});
