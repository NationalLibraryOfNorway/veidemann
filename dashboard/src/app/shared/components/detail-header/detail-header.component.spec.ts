import {Component} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';

import {DetailHeaderComponent} from './detail-header.component';

@Component({
  template: `
    <app-detail-header icon="event_note" supertitle="Page log" title="https://example.org"
      titleHref="https://example.org" description="Supporting description">
      <button detailHeaderActions class="action">Action</button>
      <dl detailHeaderMetadata class="metadata"><dt>Collection</dt><dd>archive</dd></dl>
      <p detailHeaderLifecycle class="lifecycle">Running: Now</p>
    </app-detail-header>
  `,
  imports: [DetailHeaderComponent],
  standalone: true,
})
class DetailHeaderHostComponent {}

describe('DetailHeaderComponent', () => {
  let fixture: ComponentFixture<DetailHeaderHostComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [DetailHeaderHostComponent]}).compileComponents();
    fixture = TestBed.createComponent(DetailHeaderHostComponent);
    fixture.detectChanges();
  });

  it('renders the shared identity structure and all optional slots', () => {
    const header = fixture.nativeElement.querySelector('.detail-header') as HTMLElement;
    expect(header.querySelector('.detail-header-icon mat-icon')?.textContent).toBe('event_note');
    expect(header.querySelector('.detail-header-supertitle')?.textContent).toBe('Page log');
    expect(header.querySelector('h1')?.textContent.trim()).toBe('https://example.org');
    const titleLink = header.querySelector('.detail-header-title-link') as HTMLAnchorElement;
    expect(titleLink.href).toBe('https://example.org/');
    expect(titleLink.target).toBe('_blank');
    expect(titleLink.rel).toBe('noopener noreferrer');
    const heading = header.querySelector('.detail-header-heading') as HTMLElement;
    const icon = header.querySelector('.detail-header-icon') as HTMLElement;
    const description = header.querySelector('.detail-header-description') as HTMLElement;
    expect(description.textContent).toBe('Supporting description');
    expect(description.parentElement).toBe(header);
    expect(heading.lastElementChild?.tagName).toBe('H1');
    expect(getComputedStyle(header).alignItems).toBe('start');
    expect(getComputedStyle(icon).alignSelf).toBe('auto');
    expect(getComputedStyle(heading).alignSelf).toBe('auto');
    expect(getComputedStyle(header).rowGap).toBe('0px');
    expect(header.querySelector('.detail-header-actions .action')).not.toBeNull();
    expect(header.querySelector('.detail-header-metadata .metadata')).not.toBeNull();
    expect(header.querySelector('.detail-header-lifecycle .lifecycle')).not.toBeNull();
  });
});
