import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserconfigPreviewComponent} from './browserconfig-preview.component';
import {CommonsModule} from '../../../../commons';
import {ConfigObject, Kind} from '../../../../shared/models';
import {ShortcutListComponent} from '../../shortcut/shortcut-list/shortcut-list.component';
import {CoreTestingModule} from '../../../../core/core.testing.module';

describe('BrowserconfigPreviewComponent', () => {
  let component: BrowserconfigPreviewComponent;
  let fixture: ComponentFixture<BrowserconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CommonsModule, CoreTestingModule.forRoot()],
      declarations: [BrowserconfigPreviewComponent, ShortcutListComponent],
      providers: []
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
