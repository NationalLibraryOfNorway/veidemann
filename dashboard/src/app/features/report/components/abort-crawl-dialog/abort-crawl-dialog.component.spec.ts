import {ComponentFixture, TestBed} from '@angular/core/testing';

import {AbortCrawlDialogComponent} from './abort-crawl-dialog.component';
import {MAT_DIALOG_DATA, MatDialog, MatDialogRef} from '@angular/material/dialog';
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

  it('closes without confirming when Escape is pressed', async () => {
    const dialogRef = TestBed.inject(MatDialog).open(AbortCrawlDialogComponent, {
      data: {},
      disableClose: false,
    });
    let closed = false;
    dialogRef.afterClosed().subscribe(() => closed = true);

    const escapeEvent = new KeyboardEvent('keydown', {
      key: 'Escape',
      bubbles: true,
      cancelable: true,
    });
    Object.defineProperty(escapeEvent, 'keyCode', {get: () => 27});
    document.body.dispatchEvent(escapeEvent);
    await fixture.whenStable();

    expect(closed).toBe(true);
  });
});
