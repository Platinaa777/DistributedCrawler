import { Injectable } from '@angular/core';
import { Observable, throwError, of } from 'rxjs';
import { catchError, switchMap, tap } from 'rxjs/operators';
import { PreviewApiService } from '../../../core/services/api/preview-api.service';
import { JobCreateStateService } from './job-create-state.service';

export interface PreviewResult {
  previewId: string;
  html: string;
  url: string;
}

@Injectable({
  providedIn: 'root'
})
export class PreviewLoaderService {
  private loading = false;
  private error: string | null = null;

  constructor(
    private previewApi: PreviewApiService,
    private stateService: JobCreateStateService
  ) {}

  loadPreview(url: string): Observable<PreviewResult> {
    this.loading = true;
    this.error = null;

    return this.previewApi.createPreview(url).pipe(
      switchMap(createResponse => {
        const previewId = createResponse.id;

        // Get preview metadata with download_url
        return this.previewApi.getPreview(previewId).pipe(
          switchMap(previewResponse => {
            const preview = previewResponse.preview;

            // Fetch actual HTML from download_url
            return this.previewApi.fetchPreviewHtml(preview.download_url).pipe(
              tap(html => {
                // Update state service
                this.stateService.setPreview(url, previewId, html);
                this.loading = false;
              }),
              switchMap(html => of({
                previewId,
                html,
                url: preview.final_url || url
              }))
            );
          })
        );
      }),
      catchError(err => {
        this.loading = false;
        this.error = err.message || 'Failed to load preview';
        return throwError(() => new Error(this.error!));
      })
    );
  }

  isLoading(): boolean {
    return this.loading;
  }

  getError(): string | null {
    return this.error;
  }

  clearError(): void {
    this.error = null;
  }
}
