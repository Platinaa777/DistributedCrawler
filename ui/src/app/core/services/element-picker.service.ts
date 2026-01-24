import { Injectable } from '@angular/core';
import { Subject, Observable } from 'rxjs';

export interface SelectedElement {
  selector: string;
  value: string;
  attribute: string;
  element: Element;
}

@Injectable({
  providedIn: 'root'
})
export class ElementPickerService {
  private selectedElement$ = new Subject<SelectedElement>();

  selectElement(data: SelectedElement): void {
    this.selectedElement$.next(data);
  }

  getSelectedElement(): Observable<SelectedElement> {
    return this.selectedElement$.asObservable();
  }
}
