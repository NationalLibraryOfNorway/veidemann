import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ErrorDialogComponent} from './error-dialog.component';
import {provideZonelessChangeDetection} from '@angular/core';
import {ComponentFixture, TestBed} from '@angular/core/testing';

describe('ErrorDialogComponent', () => {
  let fixture: ComponentFixture<ErrorDialogComponent>;
  let component: ErrorDialogComponent;


  const EXPECTED_DIALOG = {
    error: {
      name: 'My Error',
      message: 'My Errormessage',
      code: -3
    }
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [ErrorDialogComponent],
      providers: [
        provideZonelessChangeDetection(),
        {provide: MatDialogRef, useValue: {}},
        {provide: MAT_DIALOG_DATA, useValue: EXPECTED_DIALOG},
      ]
    });

    fixture = TestBed.createComponent(ErrorDialogComponent);
    component = fixture.componentInstance;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should have title set', () => {
    expect(component.title).toBe(EXPECTED_DIALOG.error.name);
  });

  it('should have content set', () => {
    expect(component.content).toBe(EXPECTED_DIALOG.error.message);
  });
});
