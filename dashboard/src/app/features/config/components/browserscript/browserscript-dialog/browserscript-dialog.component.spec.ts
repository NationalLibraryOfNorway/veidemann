import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserScriptDialogComponent} from './browserscript-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {describe, it } from 'vitest'
import {createMonacoEditorTestingHarness} from '../../../../../shared/components/monaco-editor/monaco-editor.spec-helpers';

describe('BrowserScriptDialogComponent', () => {
  let component: BrowserScriptDialogComponent;
  let fixture: ComponentFixture<BrowserScriptDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.BROWSERSCRIPT}),
    options: {}
  };

  beforeEach(() => {
    const monaco = createMonacoEditorTestingHarness();
    TestBed.configureTestingModule({
      imports: [
        BrowserScriptDialogComponent,
      ],
      providers: [
        ...provideCoreTesting,
        monaco.provider,
        {
          provide: MAT_DIALOG_DATA,
          useValue: MY_CONF
        },
        {
          provide: MatDialogRef,
          useValue: {}
        }
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserScriptDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
