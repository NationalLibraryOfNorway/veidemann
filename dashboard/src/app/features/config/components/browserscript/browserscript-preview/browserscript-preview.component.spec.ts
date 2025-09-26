import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserscriptPreviewComponent} from './browserscript-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {MonacoEditorModule} from 'ngx-monaco-editor-v2';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('BrowserscriptPreviewComponent', () => {
  let component: BrowserscriptPreviewComponent;
  let fixture: ComponentFixture<BrowserscriptPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        BrowserscriptPreviewComponent,
        MonacoEditorModule.forRoot()
      ],
      providers: [
        ...provideCoreTesting
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserscriptPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({ kind: Kind.BROWSERSCRIPT });
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
