import { Injectable } from '@angular/core';

@Injectable({
  providedIn: 'root'
})
export class SelectorGeneratorService {
  /**
   * Generates a unique CSS selector for a DOM element
   * Strategy: Use ID if available, else build path with nth-child
   */
  generate(element: Element): string {
    // 1. If element has ID, use it
    if (element.id) {
      return `#${CSS.escape(element.id)}`;
    }

    // 2. Build path from element to root
    const path: string[] = [];
    let current: Element | null = element;

    while (current && current !== current.ownerDocument?.documentElement) {
      let selector = current.tagName.toLowerCase();

      // Add nth-child if needed for uniqueness
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

  /**
   * Extract text or attribute from element
   */
  extractValue(element: Element, attribute: string): string {
    if (attribute === 'text') {
      return element.textContent?.trim() || '';
    }
    return element.getAttribute(attribute) || '';
  }
}
