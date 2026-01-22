# Summary for designer (Context7 + PrimeNG v19)

Goal: refresh the frontend visuals to a more modern look using Context7 and PrimeNG v19 components/themes.

## Project scope
- Frontend is in `ui/` (Angular 19, standalone components).
- Current UI uses Angular Material + Tailwind utilities; templates mostly inline in `*.component.ts`.
- Global styles: `ui/src/styles.scss` (Tailwind + Roboto).
- Fonts/icons currently: Roboto + Material Icons via `ui/src/index.html`.

## Navigation & roles
Routes live in `ui/src/app/app.routes.ts`.

Pages:
- /auth/login
- /auth/register
- /jobs (main list)
- /jobs/simple-create (simple form)
- /jobs/create (wizard)
- /jobs/:id (details)
- /workers (admin)
- /users (admin)

Roles:
- READ
- READ_WRITE
- ADMINISTRATOR

## Core screens and UX patterns
1) Auth (login/register)
   - Fullscreen gradient background.
   - Card-based form with input fields, password toggle, error block, loading spinner.
   - Calls to action and helper links.

2) Jobs list
   - Header with title and actions (Create Job, Users).
   - Filter panel (name/status/date range) + clear button.
   - Table with expandable row (detail view shows JSON).
   - Empty state + loading + error cards.
   - "Load more" pagination button.

3) Job details
   - Summary card with status, created date, depth, rate limit.
   - Chips/tags for seeds and export status.
   - Export buttons for JSON/CSV with loading indicator.
   - Task table with paginator and download actions.

4) Job creation
   - Simple form (single long page) with sections and dynamic arrays (seeds/domains/fields/metrics/pagination).
   - Wizard (4 steps): URL preview, element picker, settings, review & create.
   - Element picker: iframe preview + hover overlay, field/metric builders, mock JSON preview.

5) Worker health (admin)
   - KPI cards (total/active/draining/inactive).
   - Worker table with status tags.
   - Refresh + auto polling.

6) Users (admin)
   - Table with role selection and update spinners.
   - Back-to-jobs action.

## Components mapping (Material -> PrimeNG)
- Card: MatCard -> p-card
- Table: MatTable -> p-table
- Chips: MatChip -> p-tag
- Form fields: MatFormField/MatInput -> p-inputText, p-password, p-inputNumber, p-textarea
- Select: MatSelect -> p-dropdown
- Buttons: MatButton -> p-button
- Menu: MatMenu -> p-menu or p-tieredMenu
- Stepper: MatStepper -> p-stepper/p-steps
- Spinner: MatProgressSpinner -> p-progressSpinner
- Paginator: MatPaginator -> p-paginator
- Snackbar: MatSnackBar -> p-toast

## Notes for visual redesign
- Many pages use card + table + empty state patterns; keep structure but refresh styling.
- Jobs list and details have JSON blocks; ensure monospace and overflow styles.
- Element picker relies on an iframe and overlay; design should preserve a clear focus on the preview area.
- Role-based actions: create, users, workers are gated; ensure button prominence matches role level.

## Locations to review
- `ui/src/app/features/auth/*`
- `ui/src/app/features/jobs/*`
- `ui/src/app/features/job-details/*`
- `ui/src/app/features/job-create/*`
- `ui/src/app/features/workers/*`
- `ui/src/app/features/users/*`
- `ui/src/app/app.component.html`
- `ui/src/styles.scss`, `ui/src/index.html`
