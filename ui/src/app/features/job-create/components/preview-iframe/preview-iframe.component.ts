import { Component, Input, Output, EventEmitter, ViewChild, ElementRef, AfterViewInit } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-preview-iframe',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="relative w-full h-full border border-gray-300 rounded-lg overflow-hidden bg-white">
      <iframe
        #previewFrame
        sandbox="allow-same-origin"
        [srcdoc]="html || ''"
        class="w-full h-full"
        (load)="onIframeLoad()"
      ></iframe>

      <div *ngIf="!html" class="absolute inset-0 flex items-center justify-center bg-gray-50">
        <p class="text-gray-500">No preview loaded</p>
      </div>
    </div>
  `,
  styles: [`
    :host {
      display: block;
      width: 100%;
      height: 100%;
    }

    iframe {
      border: none;
      background: white;
    }
  `]
})
export class PreviewIframeComponent implements AfterViewInit {
  @Input() html: string | null = null;
  @Output() frameReady = new EventEmitter<HTMLIFrameElement>();

  @ViewChild('previewFrame') previewFrame!: ElementRef<HTMLIFrameElement>;

  ngAfterViewInit(): void {
    // Emit iframe reference after view init
    if (this.previewFrame) {
      setTimeout(() => {
        this.frameReady.emit(this.previewFrame.nativeElement);
      }, 100);
    }
  }

  onIframeLoad(): void {
    // Emit iframe reference when loaded
    if (this.previewFrame) {
      this.frameReady.emit(this.previewFrame.nativeElement);
    }
  }

  getIframeDocument(): Document | null {
    if (!this.previewFrame) {
      return null;
    }

    const iframe = this.previewFrame.nativeElement;
    return iframe.contentDocument || iframe.contentWindow?.document || null;
  }

  getIframeElement(): HTMLIFrameElement | null {
    return this.previewFrame?.nativeElement || null;
  }
}
