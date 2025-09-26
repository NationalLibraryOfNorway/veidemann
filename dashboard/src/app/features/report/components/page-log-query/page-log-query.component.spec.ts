import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PageLogQueryComponent} from './page-log-query.component';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';

describe('PageLogQueryComponent', () => {
  let component: PageLogQueryComponent;
  let fixture: ComponentFixture<PageLogQueryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      declarations: [PageLogQueryComponent],
      providers: [
        provideZonelessChangeDetection() // Use zoneless change detection for better performance in tests
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogQueryComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
