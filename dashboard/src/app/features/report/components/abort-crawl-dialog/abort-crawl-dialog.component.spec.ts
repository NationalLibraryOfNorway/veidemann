import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AbortCrawlDialogComponent} from './abort-crawl-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import { provideZonelessChangeDetection } from '@angular/core';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('AbortCrawlDialogComponent', () => {
  let component: AbortCrawlDialogComponent;
  let fixture: ComponentFixture<AbortCrawlDialogComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [AbortCrawlDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: {}},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(AbortCrawlDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
