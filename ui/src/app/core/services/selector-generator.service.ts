import { Injectable } from '@angular/core';
import { finder } from '@medv/finder';

@Injectable({
  providedIn: 'root'
})
export class SelectorGeneratorService {
  /**
   * Generates a unique CSS selector for a DOM element
   */
  generate(element: Element): string {
    try {
      return finder(element);
    } catch {
      return this.generateFallback(element);
    }
  }

  /**
   * Extract text or attribute from element
   */
  extractValue(element: Element, attribute: string): string {
    if (attribute === 'text') {
      return element.textContent?.trim() || '';
    }
    return element.getAttribute(attribute) || '';
  }

  private generateFallback(element: Element): string {
    if (element.id) {
      return `#${CSS.escape(element.id)}`;
    }

    const path: string[] = [];
    let current: Element | null = element;

    while (current && current !== current.ownerDocument?.documentElement) {
      let selector = current.tagName.toLowerCase();
      const parent: Element | null = current.parentElement;
      if (parent) {
        const currentTagName = current.tagName;
        const siblings = Array.from(parent.children).filter(
          (el: Element) => el.tagName === currentTagName
        );
        if (siblings.length > 1) {
          const index = siblings.indexOf(current) + 1;
          selector += `:nth-child(${index})`;
        }
      }

      path.unshift(selector);
      current = parent;
    }

    return path.join(' > ');
  }
}
