import { Routes } from '@angular/router';
import { authGuard } from './core/guards/auth.guard';
import { roleGuard } from './core/guards/role.guard';

export const routes: Routes = [
  {
    path: '',
    redirectTo: '/jobs',
    pathMatch: 'full'
  },
  {
    path: 'auth',
    children: [
      {
        path: 'login',
        loadComponent: () => import('./features/auth/login.component').then(m => m.LoginComponent)
      },
      {
        path: 'register',
        loadComponent: () => import('./features/auth/register.component').then(m => m.RegisterComponent)
      }
    ]
  },
  {
    path: 'jobs',
    canActivate: [authGuard],
    loadComponent: () => import('./features/jobs/jobs-list.component').then(m => m.JobsListComponent)
  },
  {
    path: 'jobs/simple-create',
    canActivate: [authGuard, roleGuard],
    data: { minRole: 'READ_WRITE' },
    loadComponent: () => import('./features/job-create/simple-job-create.component').then(m => m.SimpleJobCreateComponent)
  },
  {
    path: 'jobs/create',
    canActivate: [authGuard, roleGuard],
    data: { minRole: 'READ_WRITE' },
    loadComponent: () => import('./features/job-create/job-create.component').then(m => m.JobCreateComponent)
  },
  {
    path: 'jobs/:id',
    canActivate: [authGuard],
    loadComponent: () => import('./features/job-details/job-details.component').then(m => m.JobDetailsComponent)
  },
  {
    path: 'workers',
    canActivate: [authGuard, roleGuard],
    data: { minRole: 'ADMINISTRATOR' },
    loadComponent: () => import('./features/workers/worker-monitor.component').then(m => m.WorkerMonitorComponent)
  },
  {
    path: 'users',
    canActivate: [authGuard, roleGuard],
    data: { minRole: 'ADMINISTRATOR' },
    loadComponent: () => import('./features/users').then(m => m.UsersListComponent)
  },
  {
    path: '**',
    redirectTo: '/jobs'
  }
];
