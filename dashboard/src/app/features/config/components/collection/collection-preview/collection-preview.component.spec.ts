import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CollectionPreviewComponent} from './collection-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CollectionPreviewComponent', () => {
  let component: CollectionPreviewComponent;
  let fixture: ComponentFixture<CollectionPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CollectionPreviewComponent],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CollectionPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.COLLECTION});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

