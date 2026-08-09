import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ConfigLabelLinksComponent} from './config-label-links.component';

describe('ConfigLabelLinksComponent', () => {
  let fixture: ComponentFixture<ConfigLabelLinksComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfigLabelLinksComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();
    fixture = TestBed.createComponent(ConfigLabelLinksComponent);
  });

  it('renders each link as a complete list row that opens in a new tab', () => {
    fixture.componentRef.setInput('links', [{text: 'Owner registry', href: 'https://example.com/owner/one'}]);
    fixture.detectChanges();

    const row = fixture.nativeElement.querySelector('a[mat-list-item]') as HTMLAnchorElement;
    expect(row.querySelector('[matListItemTitle]').textContent.trim()).toBe('Owner registry');
    expect(row.target).toBe('_blank');
    expect(row.rel).toBe('noopener noreferrer');
    expect(row.querySelector('mat-icon')).toBeNull();
    expect(getComputedStyle(row).cursor).toBe('pointer');
  });
});
