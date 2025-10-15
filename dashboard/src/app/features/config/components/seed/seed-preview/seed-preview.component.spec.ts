import {ComponentFixture, TestBed} from '@angular/core/testing';

import {SeedPreviewComponent} from './seed-preview.component';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';
import {of} from 'rxjs';

describe('SeedPreviewComponent', () => {
  let component: SeedPreviewComponent;
  let fixture: ComponentFixture<SeedPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [SeedPreviewComponent],
      providers: [
        ...provideCoreTesting,
        { provide: ActivatedRoute, useValue: { snapshot: {}, params: of({}), queryParams: of({}) } }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(SeedPreviewComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
