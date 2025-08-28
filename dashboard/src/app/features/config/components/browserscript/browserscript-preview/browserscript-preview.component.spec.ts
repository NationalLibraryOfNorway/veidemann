import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideZonelessChangeDetection } from '@angular/core';

import { BrowserscriptPreviewComponent } from './browserscript-preview.component';
import { CommonsModule } from '../../../../commons';
import { ConfigObject, Kind } from '../../../../shared/models';
import { MonacoEditorModule } from 'ngx-monaco-editor-v2';

describe('BrowserscriptPreviewComponent', () => {
  let component: BrowserscriptPreviewComponent;
  let fixture: ComponentFixture<BrowserscriptPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CommonsModule,
        MonacoEditorModule.forRoot()
      ],
      declarations: [BrowserscriptPreviewComponent],
      providers: [
        provideZonelessChangeDetection()
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
