import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: '',
    redirectTo: '/jobs',
    pathMatch: 'full'
  },
  {
    path: 'jobs',
    loadComponent: () => import('./features/jobs/jobs-list.component').then(m => m.JobsListComponent)
  },
  {
    path: 'jobs/simple-create',
    loadComponent: () => import('./features/job-create/simple-job-create.component').then(m => m.SimpleJobCreateComponent)
  },
  {
    path: 'jobs/create',
    loadComponent: () => import('./features/job-create/job-create.component').then(m => m.JobCreateComponent)
  },
  {
    path: 'jobs/:id',
    loadComponent: () => import('./features/job-details/job-details.component').then(m => m.JobDetailsComponent)
  }
];
