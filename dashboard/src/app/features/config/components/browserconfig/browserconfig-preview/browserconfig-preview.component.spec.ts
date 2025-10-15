import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserconfigPreviewComponent} from './browserconfig-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('BrowserconfigPreviewComponent', () => {
  let component: BrowserconfigPreviewComponent;
  let fixture: ComponentFixture<BrowserconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports:[BrowserconfigPreviewComponent],
      declarations: [],
      providers: [
        ...provideCoreTesting
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserconfigPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.BROWSERCONFIG});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
