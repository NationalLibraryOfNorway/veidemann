import {ComponentFixture, TestBed} from '@angular/core/testing';
import {PreviewComponent} from './preview.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {ActivatedRoute} from '@angular/router';
import {of} from 'rxjs';

describe('PreviewComponent', () => {
  let component: PreviewComponent;
  let fixture: ComponentFixture<PreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PreviewComponent],
      providers: [
        ...provideCoreTesting,
        { provide: ActivatedRoute, useValue: { snapshot: {}, params: of({}), queryParams: of({}) } }
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PreviewComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
