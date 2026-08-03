import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideRouter} from '@angular/router';

import {SectionHeaderComponent} from './section-header.component';

describe('SectionHeaderComponent', () => {
  let fixture: ComponentFixture<SectionHeaderComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SectionHeaderComponent],
      providers: [provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(SectionHeaderComponent);
    fixture.componentRef.setInput('rootLabel', 'Configuration');
    fixture.componentRef.setInput('rootLink', ['/config']);
    fixture.componentRef.setInput('title', 'Seed');
    fixture.componentRef.setInput('backLink', ['/config/seed']);
    fixture.componentRef.setInput('listLink', ['/config/seed']);
    fixture.componentRef.setInput('detail', true);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('uses deterministic parent links for detail navigation', () => {
    const back = fixture.nativeElement.querySelector('.back-button');
    const breadcrumbs = fixture.nativeElement.querySelectorAll('.breadcrumbs a');

    expect(back.getAttribute('href')).toBe('/config/seed');
    expect(breadcrumbs[0].getAttribute('href')).toBe('/config');
    expect(breadcrumbs[1].getAttribute('href')).toBe('/config/seed');
  });
});
